package executor

import (
	"context"
	"fmt"
	"net"
	"os/user"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/retry"
	"golang.org/x/crypto/ssh"

	"github.com/sapslaj/mid/agent"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/provider/types"
)

func DialWithRetry[T any](ctx context.Context, msg string, maxAttempts int, f func() (T, error)) (T, error) {
	var userError error
	ok, data, err := retry.Until(ctx, retry.Acceptor{
		Accept: func(try int, _ time.Duration) (bool, any, error) {
			var result T
			result, userError = f()
			if userError == nil {
				return true, result, nil
			}
			dials := try + 1
			if maxAttempts > -1 && dials > maxAttempts {
				return true, nil, fmt.Errorf(
					"after %d failed attempts: %w",
					try,
					userError,
				)
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
		return data.(T), nil
	}

	var t T
	if err == nil {
		err = ctx.Err()
	}
	return t, err
}

func StartAgent(ctx context.Context, connection *types.Connection) (*agent.Agent, error) {
	username := "root"
	if connection.User == nil {
		current, err := user.Current()
		if err == nil {
			username = current.Username
		}
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
			return nil, err
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

	sshClient, err := DialWithRetry(ctx, "Dial", 10, func() (*ssh.Client, error) {
		return ssh.Dial("tcp", endpoint, sshConfig)
	})
	if err != nil {
		return nil, err
	}

	a := &agent.Agent{
		Client: sshClient,
	}

	err = agent.Connect(a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func CallAgent[I any, O any](a *agent.Agent, call rpc.RPCCall[I]) (rpc.RPCResult[O], error) {
	return agent.Call[I, O](a, call)
}
