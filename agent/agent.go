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
	"golang.org/x/crypto/ssh"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/syncmap"
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

type Agent struct {
	Mutex     sync.Mutex
	Client    *ssh.Client
	Session   *ssh.Session
	Encoder   *json.Encoder
	Decoder   *json.Decoder
	Running   *atomic.Bool
	WaitGroup *sync.WaitGroup
	InFlight  *syncmap.Map[string, chan rpc.RPCResult[any]]
	Logger    *slog.Logger
}

func AddLogger(agent *Agent) *slog.Logger {
	if agent.Logger == nil {
		agent.Logger = slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelDebug,
				},
			),
		).With(slog.String("side", "local"))
	}

	return agent.Logger
}

func RunRemoteCommand(agent *Agent, cmd string) ([]byte, error) {
	session, err := agent.Client.NewSession()
	if err != nil {
		return nil, errors.Join(ErrRunningRemoteCommand, err)
	}
	defer session.Close()
	return session.Output(cmd)
}

func InstallAgent(agent *Agent) error {
	initOutput, err := RunRemoteCommand(agent, "touch .mid/install.lock && uname -m")
	if err != nil {
		return err
	}
	defer RunRemoteCommand(agent, "rm -f .mid/install.lock")

	// only support linux for now
	goos := "linux"
	goarch := ""
	switch strings.TrimSpace(string(initOutput)) {
	case "aarch64":
		goarch = "arm64"
	case "x86_64":
		goarch = "amd64"
	default:
		return fmt.Errorf("%w: unexpected output while initializing agent: %q", ErrInstallingAgent, string(initOutput))
	}

	agentBinary, err := GetAgentBinary(goos, goarch)
	if err != nil {
		return errors.Join(ErrInstallingAgent, err)
	}

	scpClient, err := scp.NewClientBySSH(agent.Client)
	if err != nil {
		return errors.Join(ErrInstallingAgent, err)
	}
	defer scpClient.Close()

	err = scpClient.CopyFile(context.Background(), bytes.NewReader(agentBinary), ".mid/mid-agent", "0700")
	if err != nil {
		return errors.Join(ErrInstallingAgent, err)
	}

	return nil
}

func Connect(agent *Agent) error {
	logger := AddLogger(agent)
	logger.Info("connecting agent")
	initOutput, err := RunRemoteCommand(agent, "mkdir -p .mid")
	if err != nil {
		logger.Error(
			"error creating .mid directory on remote",
			slog.Any("error", err),
			slog.String("stdout", string(initOutput)),
		)
		return errors.Join(ErrConnectingToAgent, err)
	}

	agentNotInstalled := true
	agentVersionMismatch := true

	for i := 0; i <= 10; i++ {
		if i == 10 {
			logger.Error(fmt.Sprintf("agent installation still in progress but should be finished by now; bailing"))
			return errors.Join(ErrConnectingToAgent, fmt.Errorf("another agent installation is in progress"))
		}

		// check for lock file
		installLockOutput, err := RunRemoteCommand(agent, "/bin/sh -c 'test ! -f .mid/install.lock ; echo $?'")
		if strings.TrimSpace(string(installLockOutput)) == "0" {
			break
		}

		// try it and see what happens
		initOutput, err = RunRemoteCommand(agent, "file .mid/mid-agent && .mid/mid-agent --version")
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
		err = InstallAgent(agent)
		if err != nil {
			return errors.Join(ErrConnectingToAgent, fmt.Errorf("error installing agent: %w", err))
		}
	}

	logger.Info("starting SSH session")
	agent.Session, err = agent.Client.NewSession()
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent session: %w", err))
	}

	stderr, err := agent.Session.StderrPipe()
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent stderr pipe: %w", err))
	}
	go io.Copy(os.Stdout, stderr)

	stdin, err := agent.Session.StdinPipe()
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent stdin pipe: %w", err))
	}
	stdout, err := agent.Session.StdoutPipe()
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error creating agent stdout pipe: %w", err))
	}
	agent.Encoder = json.NewEncoder(stdin)
	agent.Decoder = json.NewDecoder(stdout)

	logger.Info("starting agent")

	// for some reason Ansible doesn't like Docker containers with sudo installed
	// so have to jump through some hoops to not use sudo if we don't have to.
	idOutput, err := RunRemoteCommand(agent, "id")
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error getting UID: %w", err))
	}
	useSudo := true
	if strings.Contains(string(idOutput), "uid=0(") {
		useSudo = false
	}

	sessionStartCmd := ""
	if useSudo {
		sessionStartCmd += "sudo "
	}
	sessionStartCmd += ".mid/mid-agent"

	// TODO: more extensible sudo configuration
	err = agent.Session.Start(sessionStartCmd)
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error starting agent session: %w", err))
	}

	agent.Running = &atomic.Bool{}
	agent.Running.Store(true)
	agent.WaitGroup = &sync.WaitGroup{}
	agent.WaitGroup.Add(1)
	agent.InFlight = &syncmap.Map[string, chan rpc.RPCResult[any]]{}

	go func() {
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
		defer logger.Info("shutting down decoder loop")
		for agent.Running.Load() {
			logger.Info("waiting for next result")
			var res rpc.RPCResult[any]
			err = agent.Decoder.Decode(&res)
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

			decoderLogger.Info("got result")

			ch, loaded := agent.InFlight.LoadAndDelete(res.UUID)
			if !loaded {
				decoderLogger.Warn("UUID not found in InFlight map")
			}
			if ch == nil {
				decoderLogger.Error("result channel is nil, cannot send result")
				continue
			}

			decoderLogger.Info("channeling result")
			ch <- res
		}
	}()

	logger.Info("pinging agent")
	pingResult, err := Call[rpc.AgentPingArgs, rpc.AgentPingResult](agent, rpc.RPCCall[rpc.AgentPingArgs]{
		RPCFunction: rpc.RPCAgentPing,
		Args: rpc.AgentPingArgs{
			Ping: "ping",
		},
	})
	if err != nil {
		return errors.Join(ErrConnectingToAgent, fmt.Errorf("error sending ping RPC: %w", err))
	}
	if pingResult.Error != "" {
		return errors.Join(
			ErrConnectingToAgent,
			fmt.Errorf("error received from ping RPC: %w", errors.New(pingResult.Error)),
		)
	}

	return nil
}

func Call[I any, O any](agent *Agent, call rpc.RPCCall[I]) (rpc.RPCResult[O], error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		return rpc.RPCResult[O]{
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}
	call.UUID = uuid.String()

	agent.Mutex.Lock()
	err = agent.Encoder.Encode(call)
	agent.Mutex.Unlock()
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		return rpc.RPCResult[O]{
			RPCFunction: call.RPCFunction,
			Error:       err.Error(),
		}, err
	}

	// special case for "Close" since no response is expected
	if call.RPCFunction == rpc.RPCClose {
		return rpc.RPCResult[O]{
			UUID:        call.UUID,
			RPCFunction: call.RPCFunction,
		}, nil
	}

	ch := make(chan rpc.RPCResult[any])
	agent.InFlight.Store(call.UUID, ch)

	rawResult := <-ch
	res, err := rpc.AnyToJSONT[rpc.RPCResult[O]](rawResult)
	if err != nil {
		err = errors.Join(ErrCallingRPCSystem, err)
		if res.Error == "" {
			res.Error = err.Error()
		} else {
			res.Error = errors.Join(errors.New(res.Error), err).Error()
		}
		return res, err
	}

	return res, nil
}

func StageFile(agent *Agent, f io.Reader) (string, error) {
	_, err := RunRemoteCommand(agent, "mkdir -p .mid/staging")
	if err != nil {
		return "", errors.Join(ErrStagingFile, err)
	}

	scpClient, err := scp.NewClientBySSH(agent.Client)
	if err != nil {
		return "", errors.Join(ErrStagingFile, err)
	}
	defer scpClient.Close()

	uid, err := uuid.NewRandom()
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		return "", err
	}
	remotePath := "mid/staging/" + strings.ToLower(uid.String())

	realPathOutput, err := RunRemoteCommand(agent, "realpath "+remotePath)
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		return remotePath, err
	}

	remotePath = strings.TrimSpace(string(realPathOutput))

	err = scpClient.CopyFile(context.Background(), f, remotePath, "0400")
	if err != nil {
		err = errors.Join(ErrStagingFile, err)
		return remotePath, err
	}

	return remotePath, nil
}

func Disconnect(agent *Agent) error {
	agent.Running.Store(false)

	_, err := Call[any, any](agent, rpc.RPCCall[any]{RPCFunction: rpc.RPCClose})

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
	}

	return err
}

func (agent *Agent) Disconnect() error {
	return Disconnect(agent)
}
