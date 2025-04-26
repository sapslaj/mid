package types

type Config struct {
	Connection *Connection `pulumi:"connection" provider:"secret"`
}
