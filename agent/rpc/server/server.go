package server

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/log"
)

type Server struct {
	Logger *slog.Logger
}

func (s *Server) Start() error {
	encoder := json.NewEncoder(os.Stdout)
	decoder := json.NewDecoder(os.Stdin)
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}

	for {
		s.Logger.Info("waiting for next call")

		var err error
		var call rpc.RPCCall[any]

		err = decoder.Decode(&call)

		if err != nil {
			s.Logger.Error("error while decoding call", slog.Any("error", err))
			mutex.Lock()
			encoder.Encode(rpc.RPCResult[any]{
				UUID:  call.UUID,
				Error: err.Error(),
			})
			mutex.Unlock()
			continue
		}

		if call.UUID == "" {
			s.Logger.Error(
				"UUID is empty",
				slog.Any("name", call.RPCFunction),
				log.SlogJSON("args", call.Args),
			)
			mutex.Lock()
			encoder.Encode(rpc.RPCResult[any]{
				UUID:  call.UUID,
				Error: "UUID is empty",
			})
			mutex.Unlock()
			continue
		}

		if call.RPCFunction == rpc.RPCClose {
			s.Logger.Info("received close, waiting for inflight to finish")
			wg.Wait()
			s.Logger.Info("closing")
			return nil
		}

		wg.Add(1)
		go func(call rpc.RPCCall[any]) {
			defer wg.Done()

			logger := s.Logger.With(
				slog.String("uuid", call.UUID),
				slog.Any("name", call.RPCFunction),
				log.SlogJSON("args", call.Args),
			)

			logger.Info("routing call")

			res, err := rpc.ServerRoute(call.RPCFunction, call.Args)

			mutex.Lock()
			defer mutex.Unlock()

			result := rpc.RPCResult[any]{
				UUID:        call.UUID,
				RPCFunction: call.RPCFunction,
				Result:      res,
			}

			if err != nil {
				logger.Error("error while handling call", slog.Any("error", err))
				result.Error = err.Error()
			}

			logger = logger.With(
				log.SlogJSON("result", res),
			)

			logger.Info("sending result")
			err = encoder.Encode(result)

			if err != nil {
				logger.Error("error while encoding result", slog.Any("error", err))
			}
		}(call)
	}
}
