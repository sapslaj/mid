package rpc

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type RPCFunction string

const (
	RPCAgentPing      RPCFunction = "AgentPing"
	RPCAnsibleExecute RPCFunction = "AnsibleExecute"
	RPCClose          RPCFunction = "Close"
	RPCExec           RPCFunction = "Exec"
	RPCFileStat       RPCFunction = "FileStat"
	RPCUntar          RPCFunction = "Untar"
)

type Server struct {
	Logger *slog.Logger
}

type RPCCall[T any] struct {
	UUID        string
	RPCFunction RPCFunction
	Args        T
}

type RPCResult[T any] struct {
	UUID        string
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

func AnyToJSONT[T any](input any) (T, error) {
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
	case RPCAgentPing:
		var targs AgentPingArgs
		targs, err := AnyToJSONT[AgentPingArgs](args)
		if err != nil {
			return nil, err
		}
		return AgentPing(targs)
	case RPCAnsibleExecute:
		var targs AnsibleExecuteArgs
		targs, err := AnyToJSONT[AnsibleExecuteArgs](args)
		if err != nil {
			return nil, err
		}
		return AnsibleExecute(targs)
	case RPCExec:
		var targs ExecArgs
		targs, err := AnyToJSONT[ExecArgs](args)
		if err != nil {
			return nil, err
		}
		return Exec(targs)
	case RPCFileStat:
		var targs FileStatArgs
		targs, err := AnyToJSONT[FileStatArgs](args)
		if err != nil {
			return nil, err
		}
		return FileStat(targs)
	}

	return nil, fmt.Errorf("unsupported RPCFunction: %s", rpcFunction)
}

func ServerStart(s *Server) error {
	encoder := json.NewEncoder(os.Stdout)
	decoder := json.NewDecoder(os.Stdin)
	mutex := sync.Mutex{}

	for {
		logger := s.Logger.With()

		s.Logger.Info("waiting for next call")

		var err error
		var call RPCCall[any]

		err = decoder.Decode(&call)

		if err != nil {
			s.Logger.Error("error while decoding call", slog.Any("error", err))
			mutex.Lock()
			encoder.Encode(RPCResult[any]{
				UUID:  call.UUID,
				Error: err.Error(),
			})
			mutex.Unlock()
			continue
		}

		if call.UUID == "" {
			s.Logger.Error("UUID is empty", slog.Any("name", call.RPCFunction), SlogJSON("args", call.Args))
			mutex.Lock()
			encoder.Encode(RPCResult[any]{
				UUID:  call.UUID,
				Error: "UUID is empty",
			})
			mutex.Unlock()
			continue
		}

		go func(call RPCCall[any]) {
			logger = logger.With(
				slog.String("uuid", call.UUID),
				slog.Any("name", call.RPCFunction),
				SlogJSON("args", call.Args),
			)

			logger.Info("routing call")

			res, err := ServerRoute(s, call.RPCFunction, call.Args)

			mutex.Lock()
			defer mutex.Unlock()

			result := RPCResult[any]{
				UUID:        call.UUID,
				RPCFunction: call.RPCFunction,
				Result:      res,
			}

			if err != nil {
				logger.Error("error while handling call", slog.Any("error", err))
				result.Error = err.Error()
			}

			logger = logger.With(
				SlogJSON("result", res),
			)

			logger.Info("sending result")
			err = encoder.Encode(result)

			if err != nil {
				logger.Error("error while encoding result", slog.Any("error", err))
			}
		}(call)
	}
}
