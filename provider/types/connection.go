package types

import "github.com/pulumi/pulumi-go-provider/infer"

type ConnectionBase struct {
	User       *string  `pulumi:"user,optional"`
	Password   *string  `pulumi:"password,optional" provider:"secret"`
	Host       *string  `pulumi:"host"`
	Port       *float64 `pulumi:"port,optional"`
	PrivateKey *string  `pulumi:"privateKey,optional" provider:"secret"`
	// TODO: add support for below
	// PrivateKeyPassword *string  `pulumi:"privateKeyPassword,optional" provider:"secret"`
	// AgentSocketPath    *string  `pulumi:"agentSocketPath,optional"`
	// DialErrorLimit     *int     `pulumi:"dialErrorLimit,optional"`
	// PerDialTimeout     *int     `pulumi:"perDialTimeout,optional"`
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
	a.SetDefault(&i.User, "root")
	a.Describe(&i.Password, "The password we should use for the connection.")
	a.Describe(&i.Host, "The address of the resource to connect to.")
	a.Describe(&i.Port, "The port to connect to. Defaults to 22.")
	a.SetDefault(&i.Port, 22)
	a.Describe(&i.PrivateKey, `The contents of an SSH key to use for the
connection. This takes preference over the password if provided.`)
}
