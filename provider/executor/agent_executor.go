package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/user"
	"sync"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/retry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/ssh"

	midagent "github.com/sapslaj/mid/agent"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/hashstructure"
	"github.com/sapslaj/mid/pkg/syncmap"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/types"
)

var (
	ErrUnreachable = errors.New("host is unreachable")
)

type ConnectionState struct {
	ID         uint64
	Reachable  bool
	Mutex      sync.Mutex
	Agent      *midagent.Agent
	Connection *types.Connection
}

var AgentPool = syncmap.Map[uint64, *ConnectionState]{}

func (cs *ConnectionState) SetupAgent(ctx context.Context) error {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.ConnectionState.SetupAgent", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *cs.Connection.Host),
	))
	defer span.End()

	cs.Mutex.Lock()
	defer cs.Mutex.Unlock()

	if cs.Agent != nil && cs.Agent.Running.Load() {
		span.SetAttributes(attribute.Bool("agent.already_running", true))
		span.SetStatus(codes.Ok, "")
		return nil
	}

	span.SetAttributes(
		attribute.Bool("agent.already_running", false),
		attribute.Bool("agent.can_connect", cs.Reachable),
	)

	a, err := StartAgent(ctx, cs.Connection)
	if err != nil {
		if errors.Is(err, midagent.ErrConnectingToAgent) {
			cs.Reachable = false
			err = errors.Join(ErrUnreachable, err)
		}
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.Bool("agent.running", false),
			attribute.Bool("agent.can_connect", cs.Reachable),
		)
		return err
	}

	cs.Reachable = true
	cs.Agent = a
	span.SetStatus(codes.Ok, "")
	span.SetAttributes(
		attribute.Bool("agent.running", true),
		attribute.Bool("agent.can_connect", cs.Reachable),
	)

	return nil
}

func Acquire(ctx context.Context, connection *types.Connection) (*ConnectionState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.Acquire", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *connection.Host),
	))
	defer span.End()

	id, err := hashstructure.Hash(connection, hashstructure.FormatV2, nil)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Float64("agent.connection_id", float64(id)))

	cs, loaded := AgentPool.LoadOrStore(id, &ConnectionState{
		ID:         id,
		Connection: connection,
	})

	span.SetAttributes(attribute.Bool("agent.loaded", loaded))
	span.SetStatus(codes.Ok, "")

	return cs, nil
}

func CanConnect(ctx context.Context, connection *types.Connection, maxAttempts int) (bool, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.CanConnect", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *connection.Host),
		attribute.Int("retry.max_attempts", maxAttempts),
	))
	defer span.End()

	cs, err := Acquire(ctx, connection)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	if cs.Reachable {
		span.SetAttributes(attribute.Bool("agent.can_connect", true))
		span.SetAttributes(attribute.Bool("agent.can_connect.cached", true))
		return true, nil
	}

	span.SetAttributes(attribute.Bool("agent.can_connect", false))
	span.SetAttributes(attribute.Bool("agent.can_connect.cached", false))

	if cs.Connection.Host == nil {
		return false, nil
	}
	if *cs.Connection.Host == "" {
		return false, nil
	}
	sshConfig, endpoint, err := ConnectionToSSHClientConfig(cs.Connection)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	sshClient, err := DialWithRetry(ctx, "Dial", maxAttempts, func() (*ssh.Client, error) {
		return ssh.Dial("tcp", endpoint, sshConfig)
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer sshClient.Close()
	session, err := sshClient.NewSession()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer session.Close()

	span.SetStatus(codes.Ok, "")
	cs.Reachable = true
	span.SetAttributes(attribute.Bool("agent.can_connect", cs.Reachable))
	return cs.Reachable, nil
}

func CallAgent[I any, O any](
	ctx context.Context,
	connection *types.Connection,
	call rpc.RPCCall[I],
) (rpc.RPCResult[O], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.CallAgent", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("rpc.function", string(call.RPCFunction)),
		telemetry.OtelJSON("rpc.args", call.Args),
	))
	defer span.End()

	var zero rpc.RPCResult[O]

	cs, err := Acquire(ctx, connection)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}

	err = cs.SetupAgent(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}

	res, err := midagent.Call[I, O](cs.Agent, call)
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	span.SetAttributes(
		attribute.String("rpc.uuid", res.UUID),
		telemetry.OtelJSON("rpc.result", res.Result),
	)

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
	connection *types.Connection,
	args I,
	preview bool,
) (O, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.AnsibleExecute", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *connection.Host),
		telemetry.OtelJSON("args", args),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	var zero O

	call, err := args.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}
	call.Args.Check = preview

	callResult, err := CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, connection, call)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return zero, err
	}

	if !callResult.Result.Success {
		maybeReturn, _ := rpc.AnyToJSONT[O](callResult.Result.Result)
		msg := maybeReturn.GetMsg()
		if msg != "" {
			err = fmt.Errorf("error running module %q: %s", call.Args.Name, msg)
		} else {
			err = fmt.Errorf(
				"error running module %q: stderr=%s stdout=%s",
				call.Args.Name,
				callResult.Result.Stderr,
				callResult.Result.Stdout,
			)
		}
		span.SetStatus(codes.Error, err.Error())
		return maybeReturn, err
	}

	returns, err := rpc.AnyToJSONT[O](callResult.Result.Result)
	if err != nil {
		err = fmt.Errorf("error decoding return value for module %q: %w", call.Args.Name, err)
		span.SetStatus(codes.Error, err.Error())
		return returns, err
	}

	span.SetStatus(codes.Ok, "")
	return returns, nil
}

func DisconnectAll(ctx context.Context) error {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.DisconnectAll", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
	))
	defer span.End()

	var err error

	for id, cs := range AgentPool.Items() {
		cs.Mutex.Lock()
		err = errors.Join(err, cs.Agent.Disconnect())
		cs.Agent = nil
		cs.Reachable = false
		cs.Mutex.Unlock()
		AgentPool.Delete(id)
	}

	return err
}

func StageFile(ctx context.Context, connection *types.Connection, f io.Reader) (string, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.StageFile")
	defer span.End()

	cs, err := Acquire(ctx, connection)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	err = cs.SetupAgent(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	remotePath, err := midagent.StageFile(cs.Agent, f)
	span.SetAttributes(attribute.String("remote_path", remotePath))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return remotePath, err
	}

	span.SetStatus(codes.Ok, "")
	return remotePath, nil
}

func ConnectionToSSHClientConfig(connection *types.Connection) (*ssh.ClientConfig, string, error) {
	username := "root"
	if connection.User == nil {
		current, err := user.Current()
		if err == nil {
			username = current.Username
		}
	} else {
		username = *connection.User
	}
	sshConfig := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second, // TODO: make this configurable
	}
	if connection.PrivateKey != nil {
		var signer ssh.Signer
		var err error
		signer, err = ssh.ParsePrivateKey([]byte(*connection.PrivateKey))
		if err != nil {
			return nil, "", err
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

	port := 22
	if connection.Port != nil {
		port = int(*connection.Port)
	}
	endpoint := net.JoinHostPort(*connection.Host, fmt.Sprintf("%d", port))
	return sshConfig, endpoint, nil
}

func DialWithRetry[T any](ctx context.Context, msg string, maxAttempts int, f func() (T, error)) (T, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.DialWithRetry", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
	))
	defer span.End()

	var userError error
	ok, data, err := retry.Until(ctx, retry.Acceptor{
		Accept: func(try int, _ time.Duration) (bool, any, error) {
			_, subspan := Tracer.Start(ctx, "mid/provider/executor.DialWithRetry:Attempt", trace.WithAttributes(
				attribute.Int("retry.attempt", try),
			))
			defer subspan.End()

			var result T
			result, userError = f()
			if userError == nil {
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
				subspan.SetStatus(codes.Error, err.Error())
				return true, nil, err
			}
			var limit string
			if maxAttempts == -1 {
				limit = "inf"
			} else {
				limit = fmt.Sprintf("%d", maxAttempts)
			}
			p.GetLogger(ctx).InfoStatusf(
				"%s %d/%s failed: retrying",
				msg,
				dials,
				limit,
			)
			return false, nil, nil
		},
	})
	if ok && err == nil {
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

func StartAgent(ctx context.Context, connection *types.Connection) (*midagent.Agent, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.StartAgent", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *connection.Host),
	))
	defer span.End()

	sshConfig, endpoint, err := ConnectionToSSHClientConfig(connection)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	sshClient, err := DialWithRetry(ctx, "Dial", 10, func() (*ssh.Client, error) {
		return ssh.Dial("tcp", endpoint, sshConfig)
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	a := &midagent.Agent{
		Client: sshClient,
	}

	err = midagent.Connect(a)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return a, nil
}
