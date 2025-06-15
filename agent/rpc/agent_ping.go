package rpc

import "os"

type AgentPingArgs struct {
	Ping string
}

type AgentPingResult struct {
	Ping string
	Pong string
	Pid  int
}

func AgentPing(args AgentPingArgs) (AgentPingResult, error) {
	return AgentPingResult{
		Ping: args.Ping,
		Pong: "pong",
		Pid:  os.Getpid(),
	}, nil
}
