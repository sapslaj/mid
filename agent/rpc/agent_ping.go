package rpc

type AgentPingArgs struct {
	Ping string
}

type AgentPingResult struct {
	Ping string
	Pong string
}

func AgentPing(args AgentPingArgs) (AgentPingResult, error) {
	return AgentPingResult{
		Ping: args.Ping,
		Pong: "pong",
	}, nil
}
