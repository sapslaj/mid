package types

import (
	"github.com/sapslaj/mid/pkg/env"
)

type Config struct {
	Connection        *Connection `pulumi:"connection" provider:"secret"`
	DeleteUnreachable bool        `pulumi:"deleteUnreachable,optional"`
}

func (config Config) GetDeleteUnreachable() bool {
	if config.DeleteUnreachable {
		return config.DeleteUnreachable
	}
	// XXX: probably don't want to panic here?
	return env.MustGetDefault("PULUMI_MID_DELETE_UNREACHABLE", false)
}
