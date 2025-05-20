package executor

import (
	"context"
	"fmt"
	"net"
	"os/user"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/retry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/ssh"

	"github.com/sapslaj/mid/agent"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/types"
)

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

func CanConnect(ctx context.Context, connection *types.Connection, maxAttempts int) (bool, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.CanConnect", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("connection.host", *connection.Host),
		attribute.Int("retry.max_attempts", maxAttempts),
	))
	defer span.End()

	if connection.Host == nil {
		return false, nil
	}
	if *connection.Host == "" {
		return false, nil
	}
	sshConfig, endpoint, err := ConnectionToSSHClientConfig(connection)
	if err != nil {
		return false, err
	}
	// TODO: adjustable maxAttempts
	sshClient, err := DialWithRetry(ctx, "Dial", maxAttempts, func() (*ssh.Client, error) {
		return ssh.Dial("tcp", endpoint, sshConfig)
	})
	if err != nil {
		return false, err
	}
	defer sshClient.Close()
	session, err := sshClient.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()
	// _, err = session.Output("echo")
	// if err != nil {
	// 	return false, err
	// }
	span.SetStatus(codes.Ok, "")
	return true, nil
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

func StartAgent(ctx context.Context, connection *types.Connection) (*agent.Agent, error) {
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

	a := &agent.Agent{
		Client: sshClient,
	}

	err = agent.Connect(a)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return a, nil
}

func CallAgent[I any, O any](ctx context.Context, a *agent.Agent, call rpc.RPCCall[I]) (rpc.RPCResult[O], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.CallAgent", trace.WithAttributes(
		attribute.String("exec.strategy", "rpc"),
		attribute.String("rpc.function", string(call.RPCFunction)),
		telemetry.OtelJSON("rpc.args", call.Args),
	))
	defer span.End()

	res, err := agent.Call[I, O](a, call)
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
