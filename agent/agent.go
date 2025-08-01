package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/ssh"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
	"github.com/sapslaj/mid/pkg/syncmap"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/version"
)

var (
	ErrRunningRemoteCommand   = errors.New("error running remote command")
	ErrInstallingAgent        = errors.New("error installing agent")
	ErrConnectingToAgent      = errors.New("error connecting to agent")
	ErrAgentShutDown          = errors.New("agent shut down")
	ErrDisconnectingFromAgent = errors.New("error disconnecting from agent")
	ErrCallingRPCSystem       = errors.New("error calling RPC system")
	ErrStagingFile            = errors.New("error staging file")
)

var Tracer = otel.Tracer("mid/agent")

type Agent struct {
	RemotePid    atomic.Int64
	InstanceUUID string
	ConnectMutex sync.Mutex
	EncoderMutex sync.Mutex
	Client       *ssh.Client
	Session      *ssh.Session
	Encoder      *json.Encoder
	Decoder      *json.Decoder
	Running      atomic.Bool
	WaitGroup    sync.WaitGroup
	InFlight     syncmap.Map[string, chan rpc.RPCResult[any]]
}

func (agent *Agent) EnsureUUID() (string, error) {
	if agent.InstanceUUID == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			return "", err
		}
		agent.InstanceUUID = uuid.String()
	}
	return agent.InstanceUUID, nil
}

func (agent *Agent) GetLogger(ctx context.Context) *slog.Logger {
	logger := telemetry.LoggerFromContext(ctx).With(slog.String("side", "local"))
	if agent.InstanceUUID != "" {
		logger = logger.With(slog.String("agent.instance.uuid", agent.InstanceUUID))
	}
	if agent.RemotePid.Load() != 0 {
		logger = logger.With(slog.Int("agent.remote.pid", int(agent.RemotePid.Load())))
	}
	return logger
}

func (agent *Agent) RunLocal() {
	ctx := context.Background()

	defer agent.WaitGroup.Done()

	agent.GetLogger(ctx).Info("starting local decoder loop")
	defer agent.GetLogger(ctx).Info("shutting down decoder loop")

	for agent.Running.Load() {
		logger := agent.GetLogger(ctx)
		logger.Debug("waiting for next result")

		var res rpc.RPCResult[any]
		err := agent.Decoder.Decode(&res)
		if err != nil {
			if !agent.Running.Load() {
				logger.Debug("got error result from decode but agent should not be running")
				return
			}

			if errors.Is(err, io.EOF) {
				logger.Debug("got EOF from decode stream")
				agent.Disconnect(context.Background(), false)
				return
			}

			if res.Error == "" {
				res.Error = errors.Join(ErrCallingRPCSystem, err).Error()
			} else {
				res.Error = errors.Join(ErrCallingRPCSystem, err, errors.New(res.Error)).Error()
			}

			logger.Error("error decoding", slog.String("error", res.Error))

			if res.UUID == "" {
				logger.Debug("result UUID is empty")
				continue
			}
		}

		logger.Debug("result appears valid, handling")

		go func(res rpc.RPCResult[any]) {
			resultLogger := agent.GetLogger(ctx).With(
				slog.Any("name", res.RPCFunction),
				telemetry.SlogJSON("result", res.Result),
				slog.String("error", res.Error),
				slog.String("rpc.uuid", res.UUID),
			)

			defer func() {
				if r := recover(); r != nil {
					resultLogger.Error("caught panic", slog.Any("error", r))
				}
			}()

			if res.UUID == "" {
				resultLogger.Error("UUID is empty")
				return
			}

			resultLogger.Debug("got result")

			for attempt := 1; attempt <= 10; attempt++ {
				ch, loaded := agent.InFlight.Load(res.UUID)
				if !loaded {
					resultLogger.Warn("UUID not found in InFlight map")
					goto retry
				}
				if ch == nil {
					resultLogger.Error("result channel is nil, cannot send result")
					goto retry
				}

				resultLogger.Debug("channeling result")
				select {
				case ch <- res:
					resultLogger.Debug("result channeled")
					return
				case <-time.After(time.Second):
					resultLogger.Warn("timed out channeling result")
					goto retry
				}

			retry:
				if attempt == 10 {
					resultLogger.Warn(fmt.Sprintf(
						"(attempt %d/10) result failed to send",
						attempt,
					))
					return
				}
				wait := time.Duration(attempt)
				resultLogger.Warn(fmt.Sprintf(
					"(attempt %d/10) result failed to send, retrying again in %s",
					attempt,
					wait,
				))
				time.Sleep(wait)
			}
		}(res)
	}
}

func (agent *Agent) Disconnect(ctx context.Context, wait bool) error {
	ctx, span := Tracer.Start(ctx, "mid/agent.Agent.Disconnect")
	defer span.End()

	alreadyStopped := !agent.Running.Load()

	agent.Running.Store(false)

	_, err := Call[any, any](ctx, agent, rpc.RPCCall[any]{RPCFunction: rpc.RPCClose})

	for uuid, ch := range agent.InFlight.Items() {
		if wait {
			func(ctx context.Context, uuid string, ch chan rpc.RPCResult[any]) {
				defer func() { recover() }()
				select {
				case ch <- rpc.RPCResult[any]{
					UUID:  uuid,
					Error: ErrAgentShutDown.Error(),
				}:
				case <-ctx.Done():
				}
			}(ctx, uuid, ch)
		} else {
			go func(uuid string, ch chan rpc.RPCResult[any]) {
				defer func() { recover() }()
				select {
				case ch <- rpc.RPCResult[any]{
					UUID:  uuid,
					Error: ErrAgentShutDown.Error(),
				}:
				case <-time.After(time.Second):
				}
			}(uuid, ch)
		}
	}

	err = errors.Join(
		err,
		agent.Session.Close(),
	)

	err = errors.Join(
		err,
		agent.Client.Close(),
	)

	if wait {
		wg := make(chan int)
		go func() {
			agent.WaitGroup.Wait()
			wg <- 0
		}()
		select {
		case <-ctx.Done():
		case <-wg:
		}
	}

	if alreadyStopped {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	if err != nil {
		err = errors.Join(ErrDisconnectingFromAgent, err)
		span.SetStatus(codes.Error, err.Error())
	}

	if err == nil {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

func (agent *Agent) Ping(ctx context.Context) (rpc.AgentPingResult, error) {
	ctx, span := Tracer.Start(ctx, "mid/agent.Agent.Ping")
	defer span.End()

	pingResult, err := Call[rpc.AgentPingArgs, rpc.AgentPingResult](ctx, agent, rpc.RPCCall[rpc.AgentPingArgs]{
		RPCFunction: rpc.RPCAgentPing,
		Args: rpc.AgentPingArgs{
			Ping: "ping",
		},
	})
	if err != nil {
		err = fmt.Errorf("error sending ping RPC: %w", err)
		span.SetStatus(codes.Error, err.Error())
		return pingResult.Result, err
	}

	if pingResult.Error != "" {
		err = fmt.Errorf("error received from ping RPC: %w", errors.New(pingResult.Error))
		span.SetStatus(codes.Error, err.Error())
		return pingResult.Result, err
	}

	return pingResult.Result, nil
}

func (agent *Agent) Heartbeat(timeout time.Duration) (time.Duration, bool) {
	ctx, span := Tracer.Start(context.Background(), "mid/agent.Agent.Heartbeat")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	logger := agent.GetLogger(ctx)
	if !agent.Running.Load() {
		logger.InfoContext(ctx, "stopping heartbeat due to agent shutdown")
		cancel()
		return 0, false
	}

	start := time.Now()

	if !agent.Running.Load() {
		cancel()
		return 0, false
	}
	_, err := agent.Ping(ctx)
	duration := time.Now().Sub(start)

	span.SetAttributes(attribute.Stringer("duration", duration))
	logger = logger.With("duration", duration)

	if !agent.Running.Load() {
		cancel()
		return duration, false
	}

	if err != nil {
		logger.ErrorContext(ctx, "failed heartbeat")
		err := agent.Disconnect(ctx, false)
		if err != nil {
			logger.ErrorContext(ctx, "error stopping", slog.Any("error", err))
		}
		cancel()
		return duration, false
	}

	logger.DebugContext(ctx, "successful heartbeat")
	cancel()
	return duration, true
}

func (agent *Agent) RunHeartbeat() {
	// TODO: make heartbeat interval configurable
	interval := time.Minute

	for {
		if !agent.Running.Load() {
			return
		}

		duration, success := agent.Heartbeat(interval)
		if !success {
			return
		}

		wait := interval - duration
		if wait > 0 {
			time.Sleep(wait)
		}
	}
}

func RunRemoteCommand(ctx context.Context, agent *Agent, cmd string) ([]byte, error) {
	ctx, span := Tracer.Start(ctx, "mid/agent.RunRemoteCommand", trace.WithAttributes(
		attribute.String("cmd", cmd),
	))
	defer span.End()

	_, err := agent.EnsureUUID()
	if err != nil {
		err = errors.Join(ErrRunningRemoteCommand, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	session, err := agent.Client.NewSession()
	if err != nil {
		err = errors.Join(ErrRunningRemoteCommand, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer session.Close()
	b, err := session.Output(cmd)
	if b != nil {
		span.SetAttributes(attribute.String("stdout", string(b)))
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return b, err
	}

	span.SetStatus(codes.Ok, "")
	return b, nil
}

func InstallAgent(ctx context.Context, agent *Agent) error {
	ctx, span := Tracer.Start(ctx, "mid/agent.InstallAgent")
	defer span.End()

	initOutput, err := RunRemoteCommand(ctx, agent, "touch .mid/install.lock && uname -m")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer RunRemoteCommand(ctx, agent, "rm -f .mid/install.lock")

	// only support linux for now
	goos := "linux"
	goarch := ""
	switch strings.TrimSpace(string(initOutput)) {
	case "aarch64":
		goarch = "arm64"
	case "x86_64":
		goarch = "amd64"
	default:
		err = fmt.Errorf("%w: unexpected output while initializing agent: %q", ErrInstallingAgent, string(initOutput))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(
		attribute.String("goos", goos),
		attribute.String("goarch", goarch),
	)

	agentBinary, err := GetAgentBinary(goos, goarch)
	if err != nil {
		err = errors.Join(ErrInstallingAgent, err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	scpClient, err := scp.NewClientBySSH(agent.Client)
	if err != nil {
		err = errors.Join(ErrInstallingAgent, err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer scpClient.Close()

	err = scpClient.CopyFile(ctx, bytes.NewReader(agentBinary), ".mid/mid-agent", "0700")
	if err != nil {
		err = errors.Join(ErrInstallingAgent, err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	stagingClean, err := RunRemoteCommand(ctx, agent, "/bin/sh -c 'if test -d .mid/staging; then rm -rf .mid/staging/*; fi'")
	span.SetAttributes(
		attribute.String("staging_clean.output", string(stagingClean)),
	)
	if err != nil {
		span.SetAttributes(
			attribute.Bool("staging_clean.success", false),
			attribute.String("staging_clean.error", err.Error()),
		)
	} else {
		span.SetAttributes(
			attribute.Bool("staging_clean.success", true),
		)
	}

	return nil
}

func Connect(ctx context.Context, agent *Agent) error {
	ctx, span := Tracer.Start(ctx, "mid/agent.Connect")
	defer span.End()

	agent.ConnectMutex.Lock()
	defer agent.ConnectMutex.Unlock()

	_, err := agent.EnsureUUID()
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	logger := agent.GetLogger(ctx)

	if agent.Running.Load() {
		logger.Warn("Connect called for already running agent")
		return nil
	}

	logger.Info("connecting agent")
	initOutput, err := RunRemoteCommand(ctx, agent, "mkdir -p .mid")
	if err != nil {
		logger.Error(
			"error creating .mid directory on remote",
			slog.Any("error", err),
			slog.String("stdout", string(initOutput)),
		)
		err = errors.Join(ErrConnectingToAgent, err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	agentNotInstalled := true
	agentVersionMismatch := true

	for i := 0; i <= 10; i++ {
		if i == 10 {
			logger.Error(fmt.Sprintf("agent installation still in progress but should be finished by now; bailing"))
			err = errors.Join(ErrConnectingToAgent, fmt.Errorf("another agent installation is in progress"))
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		// check for lock file
		installLockOutput, err := RunRemoteCommand(ctx, agent, "/bin/sh -c 'test ! -f .mid/install.lock ; echo $?'")
		if strings.TrimSpace(string(installLockOutput)) == "0" {
			break
		}

		// try it and see what happens
		initOutput, err = RunRemoteCommand(ctx, agent, "file .mid/mid-agent && .mid/mid-agent --version")
		if err == nil {
			break
		}

		// nope, installation is still going
		logger.Info(fmt.Sprintf("agent installation in progress, waiting %d seconds", i*10))
		time.Sleep(time.Duration(i) * 10 * time.Second)
	}

	agentNotInstalled = strings.Contains(string(initOutput), "No such file or directory")
	agentVersionMismatch = !strings.Contains(string(initOutput), fmt.Sprintf("mid-agent version %s", version.Version))

	if agentNotInstalled || agentVersionMismatch {
		logger.Info("copying agent")
		err = InstallAgent(ctx, agent)
		if err != nil {
			err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error installing agent: %w", err))
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	logger.Info("starting SSH session")
	agent.Session, err = agent.Client.NewSession()
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent session: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	stderr, err := agent.Session.StderrPipe()
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent stderr pipe: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	go io.Copy(os.Stdout, stderr)

	stdin, err := agent.Session.StdinPipe()
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent stdin pipe: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	stdout, err := agent.Session.StdoutPipe()
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent stdout pipe: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	agent.Encoder = json.NewEncoder(stdin)
	agent.Decoder = json.NewDecoder(stdout)

	logger.Info("starting agent")

	// for some reason Ansible doesn't like Docker containers with sudo installed
	// so have to jump through some hoops to not use sudo if we don't have to.
	idOutput, err := RunRemoteCommand(ctx, agent, "id")
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error getting UID: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	useSudo := true
	if strings.Contains(string(idOutput), "uid=0(") {
		useSudo = false
	}

	envvars := []string{}
	for _, envvar := range os.Environ() {
		if !strings.HasPrefix(envvar, "PULUMI_MID_") {
			continue
		}
		envvars = append(envvars, envvar)
	}
	envvars = append(envvars, "PULUMI_MID_AGENT_INSTANCE_UUID="+agent.InstanceUUID)

	logger.DebugContext(ctx, "passing through environment environment variables", telemetry.SlogJSON("env", envvars))

	sessionStartCmd := ""
	if len(envvars) > 0 {
		sessionStartCmd += "env "
		for _, envvar := range envvars {
			sessionStartCmd += "'"
			sessionStartCmd += envvar
			sessionStartCmd += "' "
		}
	}
	if useSudo {
		sessionStartCmd += "sudo "
		if len(envvars) > 0 {
			sessionStartCmd += "--preserve-env="
			for _, envvar := range envvars {
				parts := strings.SplitN(envvar, "=", 2)
				sessionStartCmd += parts[0]
				sessionStartCmd += ","
			}
			sessionStartCmd += " "
		}
	}
	sessionStartCmd += ".mid/mid-agent"

	logger.DebugContext(ctx, "starting session", slog.String("cmd", sessionStartCmd))

	// TODO: more extensible sudo configuration
	err = agent.Session.Start(sessionStartCmd)
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error starting agent session: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	agent.Running = atomic.Bool{}
	agent.Running.Store(true)
	agent.WaitGroup = sync.WaitGroup{}
	agent.WaitGroup.Add(1)
	agent.InFlight = syncmap.Map[string, chan rpc.RPCResult[any]]{}

	go agent.RunLocal()

	logger.Info("pinging agent")
	pingCtx, pingCancel := context.WithTimeout(ctx, time.Minute)
	defer pingCancel()
	pingResult, err := agent.Ping(pingCtx)
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error sending ping RPC: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	logger.Error("ping result", telemetry.SlogJSON("pingResult", pingResult))
	agent.RemotePid.Store(int64(pingResult.Pid))

	go agent.RunHeartbeat()

	span.SetStatus(codes.Ok, "")
	return nil
}

func Call[I any, O any](ctx context.Context, agent *Agent, call rpc.RPCCall[I]) (rpc.RPCResult[O], error) {
	ctx, span := Tracer.Start(ctx, "mid/agent.Call", trace.WithAttributes(
		telemetry.OtelJSON("rpc.call", call),
		attribute.String("rpc.function", string(call.RPCFunction)),
		telemetry.OtelJSON("rpc.args", call.Args),
	))
	defer span.End()

	_, err := agent.EnsureUUID()
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		span.SetStatus(codes.Error, err.Error())
		return rpc.RPCResult[O]{
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}

	logger := agent.GetLogger(ctx).With(
		slog.String("rpc.function", string(call.RPCFunction)),
	)
	logger.DebugContext(ctx, "generating UUID for request")

	uuid, err := uuid.NewRandom()
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		span.SetStatus(codes.Error, err.Error())
		return rpc.RPCResult[O]{
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}
	call.UUID = uuid.String()
	span.SetAttributes(attribute.String("rpc.uuid", call.UUID))
	logger = logger.With(slog.String("rpc.uuid", call.UUID))
	logger.DebugContext(ctx, "generated UUID")

	logger.DebugContext(ctx, "creating result channel")
	ch := make(chan rpc.RPCResult[any])

	defer func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "caught panic closing channel", slog.Any("error", r))
			}
		}()
		logger.DebugContext(ctx, "closing result channel")
		close(ch)
	}()

	logger.DebugContext(ctx, "registering result channel as in-flight")
	agent.InFlight.Store(call.UUID, ch)

	logger.DebugContext(ctx, "acquiring encoder lock")
	agent.EncoderMutex.Lock()

	logger.DebugContext(ctx, "encoding call")
	err = agent.Encoder.Encode(call)

	logger.DebugContext(ctx, "releasing encoder lock")
	agent.EncoderMutex.Unlock()

	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		logger.ErrorContext(ctx, "error encoding", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return rpc.RPCResult[O]{
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}

	// special case for "Close" since no response is expected
	if call.RPCFunction == rpc.RPCClose {
		logger.DebugContext(ctx, "detected RPCClose, exiting early")
		span.SetStatus(codes.Ok, "")
		return rpc.RPCResult[O]{
			UUID:        call.UUID,
			RPCFunction: call.RPCFunction,
		}, nil
	}

	logger.DebugContext(ctx, "waiting for result")
	var rawResult rpc.RPCResult[any]
	select {
	case rawResult = <-ch:
		logger.DebugContext(ctx, "got result")
		break
	case <-ctx.Done():
		err := ctx.Err()
		logger.ErrorContext(ctx, "timeout waiting for result", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return rpc.RPCResult[O]{
			UUID:        call.UUID,
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}

	logger.DebugContext(ctx, "casting result to final type", telemetry.SlogJSON("rpc.raw_result", rawResult))
	span.SetAttributes(telemetry.OtelJSON("rpc.raw_result", rawResult))
	res, err := cast.AnyToJSONT[rpc.RPCResult[O]](rawResult)
	span.SetAttributes(telemetry.OtelJSON("rpc.result", res))
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		if res.Error == "" {
			res.Error = err.Error()
		} else {
			res.Error = errors.Join(errors.New(res.Error), err).Error()
		}
		logger.ErrorContext(ctx, "error casting result", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return res, err
	}

	logger.DebugContext(
		ctx,
		"got final result",
		telemetry.SlogJSON("rpc.raw_result", rawResult),
		telemetry.SlogJSON("rpc.result", res),
	)

	span.SetStatus(codes.Ok, "")
	return res, nil
}

func StageFile(ctx context.Context, agent *Agent, f io.Reader) (string, error) {
	ctx, span := Tracer.Start(ctx, "mid/agent.StageFile")
	defer span.End()

	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		if attempt == 10 {
			break
		}

		attemptCtx, attemptSpan := Tracer.Start(ctx, "mid/agent.StageFile:mkdir:Attempt", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
		))

		_, err = RunRemoteCommand(attemptCtx, agent, "mkdir -p .mid/staging")
		if err == nil {
			attemptSpan.SetStatus(codes.Ok, "")
		} else {
			attemptSpan.SetStatus(codes.Error, err.Error())
		}

		attemptSpan.End()

		if err == nil {
			break
		}

		sleepDuration := time.Duration(attempt) * 10 * time.Second
		_, sleepSpan := Tracer.Start(ctx, "mid/agent.StageFile:mkdir:Sleep", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
			attribute.String("retry.sleep", sleepDuration.String()),
		))
		time.Sleep(sleepDuration)
		sleepSpan.End()
	}
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	scpClient, err := scp.NewClientBySSH(agent.Client)
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	defer scpClient.Close()

	uid, err := uuid.NewRandom()
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	remotePath := ".mid/staging/" + strings.ToLower(uid.String())
	span.SetAttributes(attribute.String("rpc.stage_file.remote_path", remotePath))

	var realPathOutput []byte
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt == 10 {
			break
		}

		attemptCtx, attemptSpan := Tracer.Start(ctx, "mid/agent.StageFile:realpath:Attempt", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
		))

		realPathOutput, err = RunRemoteCommand(attemptCtx, agent, "realpath "+remotePath)
		if err == nil {
			attemptSpan.SetStatus(codes.Ok, "")
		} else {
			attemptSpan.SetStatus(codes.Error, err.Error())
		}

		attemptSpan.End()

		if err == nil {
			break
		}

		sleepDuration := time.Duration(attempt) * 10 * time.Second
		_, sleepSpan := Tracer.Start(ctx, "mid/agent.StageFile:realpath:Sleep", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
			attribute.String("retry.sleep", sleepDuration.String()),
		))
		time.Sleep(sleepDuration)
		sleepSpan.End()
	}
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return remotePath, err
	}

	remotePath = strings.TrimSpace(string(realPathOutput))
	span.SetAttributes(attribute.String("rpc.stage_file.absolute_remote_path", remotePath))

	for attempt := 1; attempt <= 10; attempt++ {
		if attempt == 10 {
			break
		}

		attemptCtx, attemptSpan := Tracer.Start(ctx, "mid/agent.StageFile:CopyFile:Attempt", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
		))

		err = scpClient.CopyFile(attemptCtx, f, remotePath, "0400")
		if err == nil {
			attemptSpan.SetStatus(codes.Ok, "")
		} else {
			attemptSpan.SetStatus(codes.Error, err.Error())
		}

		attemptSpan.End()

		if err == nil {
			break
		}

		sleepDuration := time.Duration(attempt) * 10 * time.Second
		_, sleepSpan := Tracer.Start(ctx, "mid/agent.StageFile:CopyFile:Sleep", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
			attribute.String("retry.sleep", sleepDuration.String()),
		))
		time.Sleep(sleepDuration)
		sleepSpan.End()
	}
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return remotePath, err
	}

	span.SetStatus(codes.Ok, "")
	return remotePath, nil
}
