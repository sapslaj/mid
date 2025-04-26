package types

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
