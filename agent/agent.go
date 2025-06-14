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
	Mutex     sync.Mutex
	Client    *ssh.Client
	Session   *ssh.Session
	Encoder   *json.Encoder
	Decoder   *json.Decoder
	Running   *atomic.Bool
	WaitGroup *sync.WaitGroup
	InFlight  *syncmap.Map[string, chan rpc.RPCResult[any]]
}

func (agent *Agent) GetLogger(ctx context.Context) *slog.Logger {
	return telemetry.LoggerFromContext(ctx).With(slog.String("side", "local"))
}

func (agent *Agent) RunLocal() {
	logger := agent.GetLogger(context.Background())
	defer agent.WaitGroup.Done()
	defer func() {
		for uuid, ch := range agent.InFlight.Items() {
			ch <- rpc.RPCResult[any]{
				UUID:  uuid,
				Error: ErrAgentShutDown.Error(),
			}
			close(ch)
		}
	}()
	logger.Info("starting local decoder loop")
	defer logger.Info("shutting down decoder loop")
	for agent.Running.Load() {
		logger.Debug("waiting for next result")
		var res rpc.RPCResult[any]
		err := agent.Decoder.Decode(&res)
		if err != nil {
			if res.Error == "" {
				res.Error = errors.Join(ErrCallingRPCSystem, err).Error()
			} else {
				res.Error = errors.Join(ErrCallingRPCSystem, err, errors.New(res.Error)).Error()
			}
			if errors.Is(err, io.EOF) && !agent.Running.Load() {
				// we're supposed to be shutting down, don't log an error
				return
			}
			logger.Error("error decoding", slog.String("error", res.Error))
			if errors.Is(err, io.EOF) {
				// not supposed to be shutting down so probably an error (hence the
				// logging above)
				return
			}
			if res.UUID == "" {
				continue
			}
		}

		decoderLogger := logger.With(
			slog.Any("name", res.RPCFunction),
			rpc.SlogJSON("result", res.Result),
			slog.String("error", res.Error),
		)

		if res.UUID == "" {
			decoderLogger.Error("UUID is empty")
			continue
		}

		decoderLogger.Debug("got result")

		ch, loaded := agent.InFlight.LoadAndDelete(res.UUID)
		if !loaded {
			decoderLogger.Warn("UUID not found in InFlight map")
		}
		if ch == nil {
			decoderLogger.Error("result channel is nil, cannot send result")
			continue
		}

		decoderLogger.Debug("channeling result")
		ch <- res
	}
}

func RunRemoteCommand(ctx context.Context, agent *Agent, cmd string) ([]byte, error) {
	ctx, span := Tracer.Start(ctx, "mid/agent.RunRemoteCommand", trace.WithAttributes(
		attribute.String("cmd", cmd),
	))
	defer span.End()

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

	return nil
}

func Connect(ctx context.Context, agent *Agent) error {
	ctx, span := Tracer.Start(ctx, "mid/agent.Connect")
	defer span.End()

	logger := agent.GetLogger(ctx)
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

	agent.Running = &atomic.Bool{}
	agent.Running.Store(true)
	agent.WaitGroup = &sync.WaitGroup{}
	agent.WaitGroup.Add(1)
	agent.InFlight = &syncmap.Map[string, chan rpc.RPCResult[any]]{}

	go agent.RunLocal()

	logger.Info("pinging agent")
	pingResult, err := Call[rpc.AgentPingArgs, rpc.AgentPingResult](ctx, agent, rpc.RPCCall[rpc.AgentPingArgs]{
		RPCFunction: rpc.RPCAgentPing,
		Args: rpc.AgentPingArgs{
			Ping: "ping",
		},
	})
	if err != nil {
		err = errors.Join(ErrConnectingToAgent, fmt.Errorf("error sending ping RPC: %w", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if pingResult.Error != "" {
		err = errors.Join(
			ErrConnectingToAgent,
			fmt.Errorf("error received from ping RPC: %w", errors.New(pingResult.Error)),
		)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

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

	agent.Mutex.Lock()
	err = agent.Encoder.Encode(call)
	agent.Mutex.Unlock()
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		span.SetStatus(codes.Error, err.Error())
		return rpc.RPCResult[O]{
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}

	// special case for "Close" since no response is expected
	if call.RPCFunction == rpc.RPCClose {
		span.SetStatus(codes.Ok, "")
		return rpc.RPCResult[O]{
			UUID:        call.UUID,
			RPCFunction: call.RPCFunction,
		}, nil
	}

	ch := make(chan rpc.RPCResult[any])
	defer close(ch)

	agent.InFlight.Store(call.UUID, ch)
	defer agent.InFlight.Delete(call.UUID)

	var rawResult rpc.RPCResult[any]
	select {
	case rawResult = <-ch:
		break
	case <-ctx.Done():
		err := ctx.Err()
		span.SetStatus(codes.Error, err.Error())
		return rpc.RPCResult[O]{
			UUID:        call.UUID,
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}
	span.SetAttributes(telemetry.OtelJSON("rpc.raw_result", rawResult))
	res, err := rpc.AnyToJSONT[rpc.RPCResult[O]](rawResult)
	span.SetAttributes(telemetry.OtelJSON("rpc.result", res))
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		if res.Error == "" {
			res.Error = err.Error()
		} else {
			res.Error = errors.Join(errors.New(res.Error), err).Error()
		}
		span.SetStatus(codes.Error, err.Error())
		return res, err
	}

	span.SetStatus(codes.Ok, "")
	return res, nil
}

func StageFile(ctx context.Context, agent *Agent, f io.Reader) (string, error) {
	ctx, span := Tracer.Start(ctx, "mid/agent.StageFile")
	defer span.End()

	_, err := RunRemoteCommand(ctx, agent, "mkdir -p .mid/staging")
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

	realPathOutput, err := RunRemoteCommand(ctx, agent, "realpath "+remotePath)
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return remotePath, err
	}

	remotePath = strings.TrimSpace(string(realPathOutput))
	span.SetAttributes(attribute.String("rpc.stage_file.absolute_remote_path", remotePath))

	err = scpClient.CopyFile(ctx, f, remotePath, "0400")
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		span.SetStatus(codes.Error, err.Error())
		return remotePath, err
	}

	span.SetStatus(codes.Ok, "")
	return remotePath, nil
}

func Disconnect(ctx context.Context, agent *Agent) error {
	ctx, span := Tracer.Start(ctx, "mid/agent.Disconnect")
	defer span.End()

	agent.Running.Store(false)

	_, err := Call[any, any](ctx, agent, rpc.RPCCall[any]{RPCFunction: rpc.RPCClose})

	err = errors.Join(
		err,
		agent.Session.Close(),
	)

	err = errors.Join(
		err,
		agent.Client.Close(),
	)

	agent.WaitGroup.Wait()

	if err != nil {
		err = errors.Join(ErrDisconnectingFromAgent, err)
		span.SetStatus(codes.Error, err.Error())
	}

	if err == nil {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

func (agent *Agent) Disconnect(ctx context.Context) error {
	ctx, span := Tracer.Start(ctx, "mid/agent.Agent.Disconnect")
	defer span.End()

	return Disconnect(ctx, agent)
}
