package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/sapslaj/mid/agent/rpc/server"
	"github.com/sapslaj/mid/pkg/log"
	"github.com/sapslaj/mid/version"
)

func main() {
	for _, arg := range os.Args {
		switch arg {
		case "--version", "-version", "-v":
			fmt.Printf("mid-agent version %s\n", version.Version)
			os.Exit(0)
		}
	}
	logfile, err := os.OpenFile(".mid-agent.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	logger := slog.New(
		slog.NewTextHandler(
			io.MultiWriter(
				logfile,
				os.Stderr,
			),
			&slog.HandlerOptions{
				AddSource: true,
				Level:     log.LogLevelFromEnv(),
			},
		),
	).With(slog.String("side", "remote"))
	defer logfile.Close()

	logger.Info("installing Ansible package")
	err = InstallAnsible()
	if err != nil {
		panic(err)
	}

	logger.Info("starting RPC server")
	defer logger.Info("stopping RPC server")
	server := &server.Server{
		Logger: logger,
	}
	err = server.Start()
	if err != nil {
		panic(err)
	}
}
