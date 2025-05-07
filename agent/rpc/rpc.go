package rpc

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

type RPCFunction string

const (
	RPCExec      RPCFunction = "Exec"
	RPCAgentPing RPCFunction = "AgentPing"
	RPCClose     RPCFunction = "Close"
)

type Server struct {
	Logger *slog.Logger
}

type RPCCall[T any] struct {
	RPCFunction RPCFunction
	Args        T
}

type RPCResult[T any] struct {
	RPCFunction RPCFunction
	Result      T
	Error       string
}

func SlogJSON(key string, value any) slog.Attr {
	data, err := json.Marshal(value)
	if err != nil {
		return slog.String(key, "err!"+err.Error())
	}
	return slog.String(key, string(data))
}

func ToArgs[T any](input any) (T, error) {
	var result T
	data, err := json.Marshal(input)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func ServerRoute(s *Server, rpcFunction RPCFunction, args any) (any, error) {
	switch rpcFunction {
	case RPCClose:
		os.Exit(0)
	case RPCExec:
		var targs ExecArgs
		targs, err := ToArgs[ExecArgs](args)
		if err != nil {
			return nil, err
		}
		return Exec(targs)
	case RPCAgentPing:
		var targs AgentPingArgs
		targs, err := ToArgs[AgentPingArgs](args)
		if err != nil {
			return nil, err
		}
		return AgentPing(targs)
	}

	return nil, fmt.Errorf("unsupported RPCFunction: %s", rpcFunction)
}

func ServerStart(s *Server) error {
	encoder := json.NewEncoder(os.Stdout)
	decoder := json.NewDecoder(os.Stdin)

	for {
		logger := s.Logger.With()

		logger.Info("waiting for next call")

		var err error
		var call RPCCall[any]

		err = decoder.Decode(&call)

		if err != nil {
			logger.Error("error while decoding call", slog.Any("error", err))
			encoder.Encode(RPCResult[any]{
				Error: err.Error(),
			})
			continue
		}

		logger = logger.With(
			slog.Any("name", call.RPCFunction),
			SlogJSON("args", call.Args),
		)

		logger.Info("routing call")

		res, err := ServerRoute(s, call.RPCFunction, call.Args)

		if err != nil {
			s.Logger.Error("error while routing call", slog.Any("error", err))
			encoder.Encode(RPCResult[any]{
				RPCFunction: call.RPCFunction,
				Result:      res,
				Error:       err.Error(),
			})
			continue
		}

		logger = logger.With(
			SlogJSON("result", res),
		)

		logger.Info("sending result")
		err = encoder.Encode(RPCResult[any]{
			RPCFunction: call.RPCFunction,
			Result:      res,
		})

		if err != nil {
			s.Logger.Error("error while encoding result", slog.Any("error", err))
		}
	}
}
