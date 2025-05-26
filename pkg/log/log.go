package log

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/sapslaj/mid/pkg/env"
)

type ContextKey string

const LoggerContextKey ContextKey = "sapslaj.mid.logger"

// LogLevelFromEnv parses the `PULUMI_MID_LOG_LEVEL` environment variable to
// get the logging level. It defaults to "INFO" if the environment variable is
// not set or if it was set to an invalid value.
func LogLevelFromEnv() slog.Level {
	s, err := env.GetDefault("PULUMI_MID_LOG_LEVEL", "INFO")
	if err != nil {
		err = fmt.Errorf("could not parse PULUMI_MID_LOG_LEVEL: %w", err)
	}
	s = strings.ToUpper(s)
	switch s {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO", "":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		err = errors.Join(err, fmt.Errorf("invalid PULUMI_MID_LOG_LEVEL: %q", s))
	}
	bootstrapLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bootstrapLogger.Error("error encountered parsing log level, defaulting to INFO", slog.Any("error", err))
	return slog.LevelInfo
}

// NewLogger builds a new slog logger which writes to stderr. We need to write
// to stderr and _not_ stdout because stdout is used for IPC stuff.
func NewLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     LogLevelFromEnv(),
			},
		),
	)
}

// ContextWithLogger stuffs the given logger (or a new logger if nil) into a
// context and returns that.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.TODO()
	}
	if logger == nil {
		logger = NewLogger()
	}
	return context.WithValue(ctx, LoggerContextKey, logger)
}

// LoggerFromContext retrieves the logger from a context previously set with
// ContextWithLogger or returns a new logger if one was not found.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		ctx = context.TODO()
	}
	logger, ok := ctx.Value(LoggerContextKey).(*slog.Logger)
	if !ok || logger == nil {
		return NewLogger()
	}
	return logger
}

// SlogJSON JSON marshals any value into a slog.Attr.
func SlogJSON(key string, value any) slog.Attr {
	data, err := json.Marshal(value)
	if err != nil {
		return slog.String(key, "err!"+err.Error())
	}
	return slog.String(key, string(data))
}
