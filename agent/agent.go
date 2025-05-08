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
	"github.com/sapslaj/mid/syncmap"
	"github.com/sapslaj/mid/version"
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
		return nil, err
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
		return fmt.Errorf("unexpected output while initializing agent: %q", string(initOutput))
	}

	agentBinary, err := GetAgentBinary(goos, goarch)
	if err != nil {
		return err
	}

	scpClient, err := scp.NewClientBySSH(agent.Client)
	if err != nil {
		return err
	}
	defer scpClient.Close()

	return scpClient.CopyFile(context.Background(), bytes.NewReader(agentBinary), ".mid/mid-agent", "0700")
}

func Connect(agent *Agent) error {
	logger := AddLogger(agent)
	logger.Info("checking if agent is cached")
	_, err := RunRemoteCommand(agent, "mkdir -p .mid")
	if err != nil {
		return err
	}

	timeout := time.Now().Add(time.Minute)
	for timeout.Sub(time.Now()) > 0 {
		lockCheck, err := RunRemoteCommand(agent, "/bin/sh -c 'test ! -f .mid/install.lock ; echo $?'")
		if err != nil {
			return err
		}
		lockCheckExit := strings.TrimSpace(string(lockCheck))
		if lockCheckExit == "0" {
			break
		}
	}

	initOutput, _ := RunRemoteCommand(agent, "file .mid/mid-agent && .mid/mid-agent --version")

	agentNotInstalled := strings.Contains(string(initOutput), "No such file or directory")
	agentVersionMismatch := !strings.Contains(string(initOutput), fmt.Sprintf("mid-agent version %s", version.Version))

	if agentNotInstalled || agentVersionMismatch {
		logger.Info("copying agent")
		err = InstallAgent(agent)
		if err != nil {
			return fmt.Errorf("error installing agent: %w", err)
		}
	}

	logger.Info("starting SSH session")
	agent.Session, err = agent.Client.NewSession()
	if err != nil {
		return fmt.Errorf("error creating agent session: %w", err)
	}

	stderr, err := agent.Session.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating agent stderr pipe: %w", err)
	}
	go io.Copy(os.Stdout, stderr)

	stdin, err := agent.Session.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating agent stdin pipe: %w", err)
	}
	stdout, err := agent.Session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating agent stdout pipe: %w", err)
	}
	agent.Encoder = json.NewEncoder(stdin)
	agent.Decoder = json.NewDecoder(stdout)

	logger.Info("starting agent")

	// for some reason Ansible doesn't like Docker containers with sudo installed
	// so have to jump through some hoops to not use sudo if we don't have to.
	idOutput, err := RunRemoteCommand(agent, "id")
	if err != nil {
		return fmt.Errorf("error getting UID: %w", err)
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
		return fmt.Errorf("error starting agent session: %w", err)
	}

	agent.Running = &atomic.Bool{}
	agent.Running.Store(true)
	agent.WaitGroup = &sync.WaitGroup{}
	agent.WaitGroup.Add(1)
	agent.InFlight = &syncmap.Map[string, chan rpc.RPCResult[any]]{}

	go func() {
		defer agent.WaitGroup.Done()
		defer logger.Info("shutting down decoder loop")
		for agent.Running.Load() {
			logger.Info("waiting for next result")
			var res rpc.RPCResult[any]
			err = agent.Decoder.Decode(&res)
			if err != nil {
				if res.Error == "" {
					res.Error = err.Error()
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
		return fmt.Errorf("error sending ping RPC: %w", err)
	}
	if pingResult.Error != "" {
		return fmt.Errorf("error received from ping RPC: %w", errors.New(pingResult.Error))
	}

	return nil
}

func Call[I any, O any](agent *Agent, call rpc.RPCCall[I]) (rpc.RPCResult[O], error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
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
		if res.Error == "" {
			res.Error = err.Error()
		}
		return res, err
	}

	return res, nil
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

	return err
}

func (agent *Agent) Disconnect() error {
	return Disconnect(agent)
}
