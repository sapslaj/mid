package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/util/retry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	midagent "github.com/sapslaj/mid/agent"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
	"github.com/sapslaj/mid/pkg/hashstructure"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/syncmap"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/midtypes"
)

var (
	ErrUnreachable = errors.New("host is unreachable")

	ErrHostUnset = errors.New("host is not set in the connection configuration")
)

type ConnectionState struct {
	ID              uint64
	Reachable       bool
	Unreachable     bool
	MaxParallel     int
	TaskCount       int
	SetupAgentMutex sync.Mutex
	CanConnectMutex sync.Mutex
	TaskCountMutex  sync.Mutex
	Agent           *midagent.Agent
	Connection      midtypes.Connection
}

var AgentPool = syncmap.Map[uint64, *ConnectionState]{}

func (cs *ConnectionState) SetupAgent(ctx context.Context) error {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.ConnectionState.SetupAgent", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *cs.Connection.Host),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With(
		slog.String("connection.host", *cs.Connection.Host),
	)

	logger.DebugContext(ctx, "SetupAgent: waiting for lock")
	p.GetLogger(ctx).InfoStatus("waiting for existing connection attempts to finish...")
	cs.SetupAgentMutex.Lock()
	logger.DebugContext(ctx, "SetupAgent: lock acquired")
	p.GetLogger(ctx).InfoStatus("") // clear info line
	defer cs.SetupAgentMutex.Unlock()

	if cs.Agent != nil && cs.Agent.Running.Load() {
		logger.With(slog.Bool("agent.already_running", true)).DebugContext(ctx, "SetupAgent: agent is already running")
		span.SetAttributes(attribute.Bool("agent.already_running", true))
		span.SetStatus(codes.Ok, "")
		return nil
	}

	span.SetAttributes(
		attribute.Bool("agent.already_running", false),
		attribute.Bool("agent.reachable", cs.Reachable),
		attribute.Bool("agent.unreachable", cs.Unreachable),
	)

	logger = logger.With(
		slog.Bool("agent.already_running", false),
		slog.Bool("agent.reachable", cs.Reachable),
		slog.Bool("agent.unreachable", cs.Unreachable),
	)

	if cs.Unreachable {
		logger.WarnContext(ctx, "SetupAgent: remote previously deemed unreachable")
		err := ErrUnreachable
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	sshConfig, endpoint, err := ConnectionToSSHClientConfig(cs.Connection)
	if err != nil {
		logger.ErrorContext(ctx, "SetupAgent: error building SSH config", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	sshClient, err := DialWithRetry(ctx, "Dial", 10, func() (*ssh.Client, error) {
		return ssh.Dial("tcp", endpoint, sshConfig)
	})
	if err != nil {
		logger.ErrorContext(ctx, "SetupAgent: error dialing", slog.Any("error", err))
		cs.Reachable = false
		cs.Unreachable = true
		span.SetStatus(codes.Error, err.Error())
		return errors.Join(ErrUnreachable, err)
	}

	cs.Agent = &midagent.Agent{
		Client: sshClient,
	}

	err = midagent.Connect(ctx, cs.Agent)
	if err != nil {
		logger.ErrorContext(ctx, "SetupAgent: error setting up agent", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	cs.Reachable = true
	span.SetAttributes(
		attribute.Bool("agent.running", true),
		attribute.Bool("agent.can_connect", cs.Reachable),
	)

	if cs.MaxParallel == 0 {
		nprocOutput, err := midagent.RunRemoteCommand(ctx, cs.Agent, "nproc")
		if err != nil {
			logger.ErrorContext(ctx, "SetupAgent: error calling nproc", slog.Any("error", err))
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		cs.MaxParallel, err = strconv.Atoi(strings.TrimSpace(string(nprocOutput)))
		if err != nil {
			logger.ErrorContext(ctx, "SetupAgent: error parsing nproc output", slog.Any("error", err))
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	logger.DebugContext(ctx, "SetupAgent: finished agent setup")
	span.SetStatus(codes.Ok, "")
	return nil
}

func (cs *ConnectionState) FinishedTask() {
	cs.TaskCountMutex.Lock()
	cs.TaskCount = max(cs.TaskCount-1, 0)
	cs.TaskCountMutex.Unlock()
}

func Acquire(
	ctx context.Context,
	connection midtypes.Connection,
	resourceConfig midtypes.ResourceConfig,
) (*ConnectionState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.Acquire", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With()

	if connection.Host == nil {
		logger.ErrorContext(ctx, "Acquire: host not set")
		err := errors.Join(ErrUnreachable, ErrHostUnset)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("connection.host", *connection.Host),
	)
	logger = logger.With(
		slog.String("connection.host", *connection.Host),
	)

	logger.DebugContext(ctx, "Acquire: calculating connection ID")
	id, err := hashstructure.Hash(connection, hashstructure.FormatV2, nil)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Float64("agent.connection_id", float64(id)))
	logger = logger.With(slog.Uint64("agent.connection_id", id))

	logger.DebugContext(ctx, "Acquire: querying pool")

	cs, loaded := AgentPool.LoadOrStore(id, &ConnectionState{
		ID:         id,
		Connection: connection,
	})

	if !loaded && cs.MaxParallel == 0 {
		cs.MaxParallel = resourceConfig.GetParallel()
	}

	logger = logger.With(slog.Bool("agent.loaded", loaded))
	span.SetAttributes(attribute.Bool("agent.loaded", loaded))

	if cs.Agent != nil && cs.MaxParallel > 0 {
		logger.DebugContext(ctx, "Acquire: MaxParallel is set, spinlocking")
		for {
			cs.TaskCountMutex.Lock()
			if cs.TaskCount <= cs.MaxParallel {
				cs.TaskCountMutex.Unlock()
				break
			}
			cs.TaskCountMutex.Unlock()
			time.Sleep(time.Millisecond)
		}
		logger.DebugContext(ctx, "Acquire: spinlock finished")
	}

	logger.DebugContext(ctx, "Acquire: returning ConnectionState handle")

	span.SetStatus(codes.Ok, "")
	return cs, nil
}

func CanConnect(
	ctx context.Context,
	connection midtypes.Connection,
	resourceConfig midtypes.ResourceConfig,
	maxAttempts int,
) (bool, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.CanConnect", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.Int("retry.max_attempts", maxAttempts),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With()

	if connection.Host == nil {
		logger.ErrorContext(ctx, "CanConnect: host not set")
		err := errors.Join(ErrUnreachable, ErrHostUnset)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	span.SetAttributes(
		attribute.String("connection.host", *connection.Host),
	)
	logger = logger.With(
		slog.String("connection.host", *connection.Host),
	)

	logger.DebugContext(ctx, "CanConnect: acquiring ConnectionState handle")
	cs, err := Acquire(ctx, connection, resourceConfig)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer cs.FinishedTask()

	logger.DebugContext(ctx, "CanConnect: waiting for lock")
	p.GetLogger(ctx).InfoStatus("waiting for existing connection attempts to finish...")
	cs.CanConnectMutex.Lock()
	logger.DebugContext(ctx, "CanConnect: lock acquired")
	p.GetLogger(ctx).InfoStatus("") // clear info line
	defer cs.CanConnectMutex.Unlock()

	if cs.Unreachable {
		span.SetAttributes(
			attribute.Bool("agent.can_connect", false),
			attribute.Bool("agent.can_connect.cached", true),
		)
		logger.With(
			slog.Bool("agent.can_connect", false),
			slog.Bool("agent.can_connect.cached", true),
		).ErrorContext(ctx, "CanConnect: remote previously deemed unreachable")
		return false, ErrUnreachable
	}

	if cs.Reachable {
		span.SetAttributes(
			attribute.Bool("agent.can_connect", true),
			attribute.Bool("agent.can_connect.cached", true),
		)
		logger.With(
			slog.Bool("agent.can_connect", true),
			slog.Bool("agent.can_connect.cached", true),
		).DebugContext(ctx, "CanConnect: remote previously deemed reachable")
		return true, nil
	}

	span.SetAttributes(
		attribute.Bool("agent.can_connect", false),
		attribute.Bool("agent.can_connect.cached", false),
	)
	logger = logger.With(slog.Bool("agent.can_connect.cached", false))

	if cs.Connection.Host == nil {
		cs.Reachable = false
		cs.Unreachable = true
		logger.With(
			slog.Bool("agent.can_connect", false),
		).ErrorContext(ctx, "CanConnect: host is nil")
		return false, nil
	}
	if *cs.Connection.Host == "" {
		cs.Reachable = false
		cs.Unreachable = true
		logger.With(
			slog.Bool("agent.can_connect", false),
		).ErrorContext(ctx, "CanConnect: host is empty")
		return false, nil
	}

	logger.DebugContext(ctx, "CanConnect: attempting connection")
	p.GetLogger(ctx).InfoStatus("attempting connection...")

	sshConfig, endpoint, err := ConnectionToSSHClientConfig(cs.Connection)
	if err != nil {
		logger.With(
			slog.Bool("agent.can_connect", false),
		).ErrorContext(ctx, "CanConnect: error building SSH config", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		cs.Reachable = false
		cs.Unreachable = true
		return false, err
	}
	sshClient, err := DialWithRetry(ctx, "Dial", maxAttempts, func() (*ssh.Client, error) {
		return ssh.Dial("tcp", endpoint, sshConfig)
	})
	if err != nil {
		logger.With(
			slog.Bool("agent.can_connect", false),
		).ErrorContext(ctx, "CanConnect: error dialing", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		cs.Reachable = false
		cs.Unreachable = true
		return false, errors.Join(ErrUnreachable, err)
	}
	defer sshClient.Close()
	session, err := sshClient.NewSession()
	if err != nil {
		logger.With(
			slog.Bool("agent.can_connect", false),
		).ErrorContext(ctx, "CanConnect: error creating agent session", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		cs.Reachable = false
		cs.Unreachable = true
		return false, errors.Join(ErrUnreachable, err)
	}
	defer session.Close()

	logger.With(
		slog.Bool("agent.can_connect", true),
	).DebugContext(ctx, "CanConnect: agent is reachable", slog.Any("error", err))
	span.SetStatus(codes.Ok, "")
	cs.Reachable = true
	cs.Unreachable = false
	span.SetAttributes(attribute.Bool("agent.can_connect", cs.Reachable))
	return cs.Reachable, nil
}

func PreviewUnreachable(
	ctx context.Context,
	connection midtypes.Connection,
	resourceConfig midtypes.ResourceConfig,
	preview bool,
) bool {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.PreviewUnreachable", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.Bool("preview", preview),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With(
		slog.Bool("preview", preview),
	)

	if connection.Host != nil {
		span.SetAttributes(
			attribute.String("connection.host", *connection.Host),
		)
		logger = logger.With(
			slog.String("connection.host", *connection.Host),
		)
	} else if preview {
		logger.WarnContext(ctx, "PreviewUnreachable: host not set")
		span.SetStatus(codes.Ok, "")
		return true
	}

	// if preview: attempt connection and return false if unreachable but true if reachable
	// if not preview: attempt connection but always return false

	connectAttempts := 10
	if preview {
		connectAttempts = 4
	}

	logger.DebugContext(
		ctx,
		fmt.Sprintf("PreviewUnreachable: using connection attempts: %d", connectAttempts),
		slog.Int("connection_attempts", connectAttempts),
	)

	canConnect, err := CanConnect(ctx, connection, resourceConfig, connectAttempts)

	span.SetAttributes(attribute.Bool("agent.can_connect", canConnect))

	if err != nil {
		span.SetAttributes(attribute.String("agent.can_connect.error", err.Error()))
	}

	span.SetStatus(codes.Ok, "")

	if canConnect {
		logger.DebugContext(ctx, "PreviewUnreachable: connection attempt succeeded")
	} else if preview {
		logger.WarnContext(ctx, "PreviewUnreachable: connection attempt failed")
	} else {
		logger.ErrorContext(ctx, "PreviewUnreachable: connection attempt failed")
	}

	if !preview {
		return false
	}

	return !canConnect
}

func CallAgent[I any, O any](
	ctx context.Context,
	connection midtypes.Connection,
	resourceConfig midtypes.ResourceConfig,
	call rpc.RPCCall[I],
) (rpc.RPCResult[O], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.CallAgent", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("rpc.function", string(call.RPCFunction)),
		telemetry.OtelJSON("rpc.args", call.Args),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With(
		slog.String("rpc.function", string(call.RPCFunction)),
	)

	logger.DebugContext(
		ctx,
		fmt.Sprintf("CallAgent: calling RPC function %q", string(call.RPCFunction)),
		telemetry.SlogJSON("call", call),
	)

	var zero rpc.RPCResult[O]

	cs, err := Acquire(ctx, connection, resourceConfig)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}
	defer cs.FinishedTask()

	if cs.Unreachable {
		err = ErrUnreachable
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}

	err = cs.SetupAgent(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}

	res, err := midagent.Call[I, O](ctx, cs.Agent, call)
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	span.SetAttributes(
		attribute.String("rpc.uuid", res.UUID),
		telemetry.OtelJSON("rpc.result", res.Result),
	)

	if res.Error != "" || err != nil {
		logger.ErrorContext(
			ctx,
			"CallAgent: got result",
			slog.Any("error", err),
			slog.String("rpc.error", res.Error),
			telemetry.SlogJSON("rpc.result", res),
		)
	} else {
		logger.DebugContext(
			ctx,
			"CallAgent: got result",
			telemetry.SlogJSON("rpc.result", res),
		)
	}

	if res.Error != "" {
		span.SetAttributes(attribute.String("rpc.error", res.Error))
	}

	return res, err
}

type AnsibleExecuteArgs interface {
	ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error)
}

type AnsibleExecuteReturn interface {
	IsChanged() bool
	GetMsg() string
}

func AnsibleExecute[I AnsibleExecuteArgs, O AnsibleExecuteReturn](
	ctx context.Context,
	connection midtypes.Connection,
	resourceConfig midtypes.ResourceConfig,
	args I,
	preview bool,
) (O, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.AnsibleExecute", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		telemetry.OtelJSON("args", args),
		attribute.Bool("preview", preview),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With(
		slog.Bool("preview", preview),
	)

	if connection.Host != nil {
		span.SetAttributes(
			attribute.String("connection.host", *connection.Host),
		)
		logger = logger.With(
			slog.String("connection.host", *connection.Host),
		)
	}

	logger.DebugContext(
		ctx,
		"AnsibleExecute: executing task",
		telemetry.SlogJSON("args", args),
	)

	var zero O

	call, err := args.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.ErrorContext(ctx, "AnsibleExecute: failed to convert args to RPC call", slog.Any("error", err))
		return zero, err
	}
	call.Args.Check = preview

	span.SetAttributes(attribute.String("ansible.name", call.Args.Name))

	if PreviewUnreachable(ctx, connection, resourceConfig, preview) {
		err = ErrUnreachable
		span.SetAttributes(attribute.Bool("unreachable", true))
		span.SetAttributes(attribute.Bool("ansible.success", false))
		span.SetStatus(codes.Error, err.Error())
		logger.WarnContext(ctx, "AnsibleExecute: bailing early due to unreachable host")
		return zero, err
	}

	callResult, err := CallAgent[
		rpc.AnsibleExecuteArgs,
		rpc.AnsibleExecuteResult,
	](
		ctx,
		connection,
		resourceConfig,
		call,
	)
	if err != nil {
		span.SetAttributes(attribute.Bool("ansible.success", false))
		span.SetStatus(codes.Error, err.Error())
		logger.ErrorContext(ctx, "AnsibleExecute: failed to call agent", slog.Any("error", err))
		return zero, err
	}

	span.SetAttributes(
		attribute.Bool("ansible.success", callResult.Result.Success),
		telemetry.OtelJSON("ansible.call_result", callResult),
	)

	if !callResult.Result.Success {
		logger.WarnContext(ctx, "AnsibleExecute: not successful, extracting error information")
		maybeReturn, maybeReturnErr := cast.AnyToJSONT[O](callResult.Result.Result)
		if maybeReturnErr != nil {
			span.SetAttributes(
				attribute.String("ansible.return.decode_error", maybeReturnErr.Error()),
			)
			logger.WarnContext(ctx, "AnsibleExecute: call result conversion failed", slog.Any("error", err))
		}

		msg := maybeReturn.GetMsg()
		if msg != "" {
			logger.DebugContext(ctx, "AnsibleExecute: using msg for error string", slog.String("msg", msg))
			err = fmt.Errorf("error running module %q: %s", call.Args.Name, msg)
		} else {
			err = fmt.Errorf(
				"error running module %q: stderr=%s stdout=%s",
				call.Args.Name,
				callResult.Result.Stderr,
				callResult.Result.Stdout,
			)
			logger.WarnContext(
				ctx,
				"AnsibleExecute: no msg found, using stderr and stdout",
				slog.String("stderr", string(callResult.Result.Stderr)),
				slog.String("stdout", string(callResult.Result.Stdout)),
			)
		}

		span.SetAttributes(
			attribute.String("ansible.msg", msg),
			telemetry.OtelJSON("ansible.return", maybeReturn),
		)
		span.SetStatus(codes.Error, err.Error())
		logger.DebugContext(
			ctx,
			"AnsibleExecute: returning errored result",
			slog.Any("error", err),
			telemetry.SlogJSON("return", maybeReturn),
		)
		return maybeReturn, err
	}

	returns, err := cast.AnyToJSONT[O](callResult.Result.Result)
	span.SetAttributes(
		telemetry.OtelJSON("ansible.return", returns),
		attribute.String("ansible.msg", returns.GetMsg()),
		attribute.Bool("ansible.is_changed", returns.IsChanged()),
	)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"AnsibleExecute: error decoding result",
			slog.Any("error", err),
			telemetry.SlogJSON("return", returns),
		)
		span.SetAttributes(
			attribute.String("ansible.return.decode_error", err.Error()),
		)
		err = fmt.Errorf("error decoding return value for module %q: %w", call.Args.Name, err)
		span.SetStatus(codes.Error, err.Error())
		return returns, err
	}

	logger.DebugContext(
		ctx,
		"AnsibleExecute: returning result",
		telemetry.SlogJSON("return", returns),
	)
	span.SetStatus(codes.Ok, "")
	return returns, nil
}

func DisconnectAll(ctx context.Context) error {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.DisconnectAll", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx)

	var multierr error

	for id, cs := range AgentPool.Items() {
		logger.DebugContext(ctx, fmt.Sprintf("DisconnectAll: disconnecting %d", id))
		cs.SetupAgentMutex.Lock()
		cs.CanConnectMutex.Lock()
		err := cs.Agent.Disconnect(ctx, true)
		multierr = errors.Join(multierr, err)
		cs.Agent = nil
		cs.Reachable = false
		cs.CanConnectMutex.Unlock()
		cs.SetupAgentMutex.Unlock()
		AgentPool.Delete(id)
		logger.DebugContext(ctx, fmt.Sprintf("DisconnectAll: disconnected %d", id), slog.Any("error", err))
	}

	logger.DebugContext(ctx, "DisconnectAll: finished disconnecting all", slog.Any("error", multierr))
	return multierr
}

func StageFile(
	ctx context.Context,
	connection midtypes.Connection,
	resourceConfig midtypes.ResourceConfig,
	f io.Reader,
) (string, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.StageFile")
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With()

	if connection.Host != nil {
		span.SetAttributes(
			attribute.String("connection.host", *connection.Host),
		)
		logger = logger.With(
			slog.String("connection.host", *connection.Host),
		)
	}

	cs, err := Acquire(ctx, connection, resourceConfig)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	defer cs.FinishedTask()

	err = cs.SetupAgent(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	logger.DebugContext(ctx, "StageFile: staging file")
	remotePath, err := midagent.StageFile(ctx, cs.Agent, f)
	span.SetAttributes(attribute.String("remote_path", remotePath))
	if err != nil {
		logger.ErrorContext(ctx, "StageFile: error staging file", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return remotePath, err
	}

	logger.DebugContext(ctx, "StageFile: finished staging file", slog.String("remote_path", remotePath))
	span.SetStatus(codes.Ok, "")
	return remotePath, nil
}

func ConnectionToSSHClientConfig(connection midtypes.Connection) (*ssh.ClientConfig, string, error) {
	sshConfig := &ssh.ClientConfig{}

	port := midtypes.DefaultConnectionPort
	if connection.Port != nil {
		port = int(*connection.Port)
	}

	if connection.Host == nil {
		return nil, "", errors.Join(ErrUnreachable, ErrHostUnset)
	}

	endpoint := net.JoinHostPort(*connection.Host, fmt.Sprintf("%d", port))

	sshConfig.User = midtypes.DefaultConnectionUser
	if connection.User == nil {
		current, err := user.Current()
		if err == nil {
			sshConfig.User = current.Username
		}
	} else {
		sshConfig.User = *connection.User
	}

	sshConfig.Timeout = time.Second * time.Duration(midtypes.DefaultConnectionPerDialTimeout)
	if connection.PerDialTimeout != nil {
		sshConfig.Timeout = time.Second * time.Duration(*connection.PerDialTimeout)
	}

	if connection.HostKey != nil {
		publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(*connection.HostKey))
		if err != nil {
			return sshConfig, endpoint, fmt.Errorf("failed to parse host key: %w", err)
		}
		sshConfig.HostKeyCallback = ssh.FixedHostKey(publicKey)
		sshConfig.HostKeyAlgorithms = []string{publicKey.Type()}
	} else {
		sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	if connection.PrivateKey != nil {
		var signer ssh.Signer
		var err error
		if connection.PrivateKeyPassword != nil {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(
				[]byte(*connection.PrivateKey),
				[]byte(*connection.PrivateKeyPassword),
			)
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(*connection.PrivateKey))
		}
		if err != nil {
			return sshConfig, endpoint, err
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	if connection.Password != nil {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(*connection.Password))
		sshConfig.Auth = append(sshConfig.Auth, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = *connection.Password
				}
				return answers, nil
			},
		))
	}

	sshAgent := false
	sshAgentSocketPath := os.Getenv("SSH_AUTH_SOCK")
	if connection.SSHAgentSocketPath != nil {
		sshAgent = true
		sshAgentSocketPath = *connection.SSHAgentSocketPath
	}
	if connection.SSHAgent != nil && *connection.SSHAgent {
		sshAgent = true
	}
	if sshAgent && sshAgentSocketPath != "" {
		agentConn, err := net.Dial("unix", sshAgentSocketPath)
		if err != nil {
			return sshConfig, endpoint, err
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
	}

	return sshConfig, endpoint, nil
}

func DialWithRetry[T any](ctx context.Context, msg string, maxAttempts int, f func() (T, error)) (T, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.DialWithRetry", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
	))
	defer span.End()

	var userError error
	ok, data, err := retry.Until(ctx, retry.Acceptor{
		// TODO: make Delay and MaxDelay configurable
		Delay:    ptr.Of(time.Second),
		MaxDelay: ptr.Of(time.Minute),
		Accept: func(try int, _ time.Duration) (bool, any, error) {
			_, subspan := Tracer.Start(ctx, "mid/provider/executor.DialWithRetry:Attempt", trace.WithAttributes(
				attribute.Int("retry.attempt", try),
			))
			defer subspan.End()
			logger := telemetry.LoggerFromContext(ctx).With(
				slog.Int("retry.attempt", try),
				slog.Int("retry.max_attempts", maxAttempts),
			)

			logger.DebugContext(ctx, "DialWithRetry.Attempt: starting attempt")

			var result T
			result, userError = f()
			if userError == nil {
				logger.DebugContext(ctx, "DialWithRetry.Attempt: success")
				subspan.SetStatus(codes.Ok, "")
				return true, result, nil
			}
			dials := try + 1
			if maxAttempts > -1 && dials > maxAttempts {
				err := fmt.Errorf(
					"after %d failed attempts: %w",
					try,
					userError,
				)
				p.GetLogger(ctx).ErrorStatus(err.Error())
				subspan.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "DialWithRetry.Attempt: giving up", slog.Any("error", err))
				return true, nil, err
			}
			var limit string
			if maxAttempts == -1 {
				limit = "inf"
			} else {
				limit = fmt.Sprintf("%d", maxAttempts)
			}
			msg := fmt.Sprintf(
				"%s %d/%s failed: retrying",
				msg,
				dials,
				limit,
			)
			subspan.SetStatus(codes.Error, msg)
			p.GetLogger(ctx).InfoStatus(msg)
			logger.DebugContext(ctx, fmt.Sprintf("DialWithRetry.Attempt: %s", msg))
			return false, nil, nil
		},
	})
	if ok && err == nil {
		p.GetLogger(ctx).InfoStatusf("%s: success", msg)
		span.SetStatus(codes.Ok, "")
		return data.(T), nil
	}

	var t T
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return t, err
	}

	span.SetStatus(codes.Ok, "")
	return t, nil
}
