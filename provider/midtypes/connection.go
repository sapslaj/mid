package midtypes

import (
	"context"

	"github.com/sapslaj/mid/pkg/providerfw/infer"
)

const (
	DefaultConnectionUser           = "root"
	DefaultConnectionPort           = 22
	DefaultConnectionPerDialTimeout = 15
)

type ConnectionBase struct {
	User               *string  `pulumi:"user,optional"`
	Password           *string  `pulumi:"password,optional" provider:"secret"`
	Host               *string  `pulumi:"host,optional"`
	Port               *float64 `pulumi:"port,optional"`
	PrivateKey         *string  `pulumi:"privateKey,optional" provider:"secret"`
	PrivateKeyPassword *string  `pulumi:"privateKeyPassword,optional" provider:"secret"`
	SSHAgent           *bool    `pulumi:"sshAgent,optional"`
	SSHAgentSocketPath *string  `pulumi:"sshAgentSocketPath,optional"`
	PerDialTimeout     *int     `pulumi:"perDialTimeout,optional"`
	HostKey            *string  `pulumi:"hostKey,optional"`
	// TODO: add support for below
	// DialErrorLimit     *int     `pulumi:"dialErrorLimit,optional"`
}

type ProxyConnection struct {
	ConnectionBase
}

type Connection struct {
	ConnectionBase
	// TODO: add support for below
	// Proxy *ProxyConnection `pulumi:"proxy,optional"`
}

func (i *Connection) Annotate(a infer.Annotator) {
	a.Describe(&i, "Instructions for how to connect to a remote endpoint.")
	a.Describe(&i.User, "The user that we should use for the connection.")
	a.SetDefault(&i.User, DefaultConnectionUser)
	a.Describe(&i.Password, "The password we should use for the connection.")
	a.Describe(&i.Host, "The address of the resource to connect to.")
	a.Describe(&i.Port, "The port to connect to. Defaults to 22.")
	a.SetDefault(&i.Port, DefaultConnectionPort)
	a.Describe(&i.PrivateKey, `The contents of an SSH key to use for the
connection. This takes preference over the password if provided.`)
}

func GetConnection(ctx context.Context, connection *Connection) Connection {
	result := Connection{}
	providerConfig := infer.GetConfig[ProviderConfig](ctx)
	if providerConfig.Connection != nil {
		result = *providerConfig.Connection
	}
	if connection != nil {
		if connection.User != nil {
			result.User = connection.User
		}
		if connection.Password != nil {
			result.Password = connection.Password
		}
		if connection.Host != nil {
			result.Host = connection.Host
		}
		if connection.PrivateKey != nil {
			result.PrivateKey = connection.PrivateKey
		}
		if connection.PrivateKeyPassword != nil {
			result.PrivateKeyPassword = connection.PrivateKeyPassword
		}
		if connection.SSHAgent != nil {
			result.SSHAgent = connection.SSHAgent
		}
		if connection.SSHAgentSocketPath != nil {
			result.SSHAgentSocketPath = connection.SSHAgentSocketPath
		}
		if connection.PerDialTimeout != nil {
			result.PerDialTimeout = connection.PerDialTimeout
		}
		if connection.HostKey != nil {
			result.HostKey = connection.HostKey
		}
	}
	return result
}
