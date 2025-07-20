package midtypes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/provider/midtypes"
)

func TestGetConnection(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		providerConfig *midtypes.ProviderConfig
		connection     *midtypes.Connection
		expect         midtypes.Connection
	}{
		"config fully from provider with nil resource connection": {
			providerConfig: &midtypes.ProviderConfig{
				Connection: &midtypes.Connection{
					ConnectionBase: midtypes.ConnectionBase{
						Host:               ptr.Of("localhost"),
						User:               ptr.Of("root"),
						Password:           ptr.Of("hunter2"),
						Port:               ptr.Of(22.0),
						PrivateKey:         ptr.Of("-----BEGIN RSA PRIVATE KEY----- ..."),
						PrivateKeyPassword: ptr.Of("anubis123"),
						SSHAgent:           ptr.Of(true),
						SSHAgentSocketPath: ptr.Of("/dev/null"),
						PerDialTimeout:     ptr.Of(20),
						HostKey:            ptr.Of("ssh-ed25519 ..."),
					},
				},
			},
			connection: nil,
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:               ptr.Of("localhost"),
					User:               ptr.Of("root"),
					Password:           ptr.Of("hunter2"),
					Port:               ptr.Of(22.0),
					PrivateKey:         ptr.Of("-----BEGIN RSA PRIVATE KEY----- ..."),
					PrivateKeyPassword: ptr.Of("anubis123"),
					SSHAgent:           ptr.Of(true),
					SSHAgentSocketPath: ptr.Of("/dev/null"),
					PerDialTimeout:     ptr.Of(20),
					HostKey:            ptr.Of("ssh-ed25519 ..."),
				},
			},
		},

		"config fully from provider with empty resource connection": {
			providerConfig: &midtypes.ProviderConfig{
				Connection: &midtypes.Connection{
					ConnectionBase: midtypes.ConnectionBase{
						Host:               ptr.Of("localhost"),
						User:               ptr.Of("root"),
						Password:           ptr.Of("hunter2"),
						Port:               ptr.Of(22.0),
						PrivateKey:         ptr.Of("-----BEGIN RSA PRIVATE KEY----- ..."),
						PrivateKeyPassword: ptr.Of("anubis123"),
						SSHAgent:           ptr.Of(true),
						SSHAgentSocketPath: ptr.Of("/dev/null"),
						PerDialTimeout:     ptr.Of(20),
						HostKey:            ptr.Of("ssh-ed25519 ..."),
					},
				},
			},
			connection: &midtypes.Connection{},
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:               ptr.Of("localhost"),
					User:               ptr.Of("root"),
					Password:           ptr.Of("hunter2"),
					Port:               ptr.Of(22.0),
					PrivateKey:         ptr.Of("-----BEGIN RSA PRIVATE KEY----- ..."),
					PrivateKeyPassword: ptr.Of("anubis123"),
					SSHAgent:           ptr.Of(true),
					SSHAgentSocketPath: ptr.Of("/dev/null"),
					PerDialTimeout:     ptr.Of(20),
					HostKey:            ptr.Of("ssh-ed25519 ..."),
				},
			},
		},

		"config fully from resource": {
			providerConfig: &midtypes.ProviderConfig{},
			connection: &midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:               ptr.Of("localhost"),
					User:               ptr.Of("root"),
					Password:           ptr.Of("hunter2"),
					Port:               ptr.Of(22.0),
					PrivateKey:         ptr.Of("-----BEGIN RSA PRIVATE KEY----- ..."),
					PrivateKeyPassword: ptr.Of("anubis123"),
					SSHAgent:           ptr.Of(true),
					SSHAgentSocketPath: ptr.Of("/dev/null"),
					PerDialTimeout:     ptr.Of(20),
					HostKey:            ptr.Of("ssh-ed25519 ..."),
				},
			},
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:               ptr.Of("localhost"),
					User:               ptr.Of("root"),
					Password:           ptr.Of("hunter2"),
					Port:               ptr.Of(22.0),
					PrivateKey:         ptr.Of("-----BEGIN RSA PRIVATE KEY----- ..."),
					PrivateKeyPassword: ptr.Of("anubis123"),
					SSHAgent:           ptr.Of(true),
					SSHAgentSocketPath: ptr.Of("/dev/null"),
					PerDialTimeout:     ptr.Of(20),
					HostKey:            ptr.Of("ssh-ed25519 ..."),
				},
			},
		},

		"partial from provider with nil resource connection": {
			providerConfig: &midtypes.ProviderConfig{
				Connection: &midtypes.Connection{
					ConnectionBase: midtypes.ConnectionBase{
						Host:     ptr.Of("localhost"),
						User:     ptr.Of("root"),
						Password: ptr.Of("hunter2"),
						Port:     ptr.Of(22.0),
					},
				},
			},
			connection: nil,
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					User:     ptr.Of("root"),
					Password: ptr.Of("hunter2"),
					Port:     ptr.Of(22.0),
				},
			},
		},

		"partial from provider with empty resource connection": {
			providerConfig: &midtypes.ProviderConfig{
				Connection: &midtypes.Connection{
					ConnectionBase: midtypes.ConnectionBase{
						Host:     ptr.Of("localhost"),
						User:     ptr.Of("root"),
						Password: ptr.Of("hunter2"),
						Port:     ptr.Of(22.0),
					},
				},
			},
			connection: &midtypes.Connection{},
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					User:     ptr.Of("root"),
					Password: ptr.Of("hunter2"),
					Port:     ptr.Of(22.0),
				},
			},
		},

		"partial from resource": {
			providerConfig: &midtypes.ProviderConfig{},
			connection: &midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					User:     ptr.Of("root"),
					Password: ptr.Of("hunter2"),
					Port:     ptr.Of(22.0),
				},
			},
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					User:     ptr.Of("root"),
					Password: ptr.Of("hunter2"),
					Port:     ptr.Of(22.0),
				},
			},
		},

		"partial from both provider and resource config": {
			providerConfig: &midtypes.ProviderConfig{
				Connection: &midtypes.Connection{
					ConnectionBase: midtypes.ConnectionBase{
						User: ptr.Of("root"),
						Port: ptr.Of(22.0),
					},
				},
			},
			connection: &midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					Password: ptr.Of("hunter2"),
				},
			},
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					User:     ptr.Of("root"),
					Password: ptr.Of("hunter2"),
					Port:     ptr.Of(22.0),
				},
			},
		},

		"resource config overrides provider config": {
			providerConfig: &midtypes.ProviderConfig{
				Connection: &midtypes.Connection{
					ConnectionBase: midtypes.ConnectionBase{
						Host:     ptr.Of("techaro.lol"),
						User:     ptr.Of("root"),
						Password: ptr.Of("anubis123"),
						Port:     ptr.Of(22.0),
					},
				},
			},
			connection: &midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					Password: ptr.Of("hunter2"),
				},
			},
			expect: midtypes.Connection{
				ConnectionBase: midtypes.ConnectionBase{
					Host:     ptr.Of("localhost"),
					User:     ptr.Of("root"),
					Password: ptr.Of("hunter2"),
					Port:     ptr.Of(22.0),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inferredConfig := infer.Config(tc.providerConfig)
			ctx := context.WithValue(context.Background(), infer.ConfigKey, inferredConfig)

			got := midtypes.GetConnection(ctx, tc.connection)

			assert.Equal(t, tc.expect, got)
		})
	}
}
