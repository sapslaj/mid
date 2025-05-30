package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/go-slog/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/sapslaj/mid/pkg/env"
	"github.com/sapslaj/mid/pkg/log"
)

// NewLogger is very similar to `log.NewLogger` except that it includes an
// otelslog handler.
func NewLogger() *slog.Logger {
	return slog.New(
		otelslog.NewHandler(
			slog.NewTextHandler(
				os.Stderr,
				&slog.HandlerOptions{
					AddSource: true,
					Level:     log.LogLevelFromEnv(),
				},
			),
		),
	)
}

// ContextWithLogger is very similar to `log.ContextWithLogger` except that it
// uses an otelslog log handler by default.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.TODO()
	}
	if logger == nil {
		logger = NewLogger()
	}
	return context.WithValue(ctx, log.LoggerContextKey, logger)
}

// LoggerFromContext is very similar to `log.LoggerFromContext` except that it
// uses an otelslog log handler by default.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		ctx = context.TODO()
	}
	logger, ok := ctx.Value(log.LoggerContextKey).(*slog.Logger)
	if !ok || logger == nil {
		return NewLogger()
	}
	return logger
}

// SlogJSON JSON marshals any value into a slog.Attr.
var SlogJSON = log.SlogJSON

type TelemetryStuff struct {
	Context        context.Context
	Logger         *slog.Logger
	CpuprofileFile *os.File
	MemprofileFile *os.File
	OtlpExporter   *otlptrace.Exporter
}

func (ts *TelemetryStuff) Shutdown() {
	ts.Logger.Debug("stopping telemetry")
	if ts.OtlpExporter != nil {
		ts.Logger.Debug("stopping OTLP exporter")
		time.Sleep(10 * time.Second)
		err := ts.OtlpExporter.Shutdown(ts.Context)
		if err != nil {
			ts.Logger.Error("error shutting down OTLP exporter", slog.Any("error", err))
		}
		ts.Logger.Debug("OTLP exporter stopped")
	}

	if ts.MemprofileFile != nil {
		ts.Logger.Debug("running final GC")
		runtime.GC()
		ts.Logger.Debug("writting allocs to memprofile file")
		err := pprof.Lookup("allocs").WriteTo(ts.MemprofileFile, 0)
		if err != nil {
			ts.Logger.Error("error writing memprofile", slog.Any("error", err))
		}
		err = ts.MemprofileFile.Close()
		if err != nil {
			ts.Logger.Error("error closing memprofile file", slog.Any("error", err))
		}
	}

	if ts.CpuprofileFile != nil {
		pprof.StopCPUProfile()
		ts.CpuprofileFile.Close()
	}
}

// StartTelemetry starts OpenTelemetry if PULUMI_MID_OTLP_ENDPOINT is set and
// returns a shutdown function. If PULUMI_MID_OTLP_ENDPOINT is not set it will
// do nothing and return an no-op function.
func StartTelemetry(ctx context.Context) *TelemetryStuff {
	// NOTE: Telemetry is _ONLY_ set up if `PULUMI_MID_OTLP_ENDPOINT` is set.
	// There is _NO_ default value for this. This means that this telemetry is
	// *OPT-IN* and only if you have an OTLP endpoint to send things to.
	// I (@sapslaj) vow to never do opt-out telemetry of my own will.

	ts := &TelemetryStuff{
		Context: ctx,
		Logger:  NewLogger(),
	}

	cpuprofilePath, err := env.GetDefault("PULUMI_MID_CPUPROFILE_PATH", "")
	if err != nil {
		ts.Logger.Error("error getting cpuprofile path", slog.Any("error", err))
	}
	if cpuprofilePath == "" {
		ts.Logger.Debug("cpu profiling disabled")
	} else {
		ts.Logger = ts.Logger.With(slog.String("cpuprofile_path", cpuprofilePath))
		ts.Logger.Info("cpu profiling enabled")
		cpuprofileFile, err := os.Create(cpuprofilePath)
		if err != nil {
			ts.Logger.Error("error opening cpuprofile path", slog.Any("error", err))
			goto memprofile
		}
		err = pprof.StartCPUProfile(cpuprofileFile)
		if err != nil {
			ts.Logger.Error("error starting CPU profiling", slog.Any("error", err))
			goto memprofile
		}
		ts.CpuprofileFile = cpuprofileFile
	}

memprofile:
	memprofilePath, err := env.GetDefault("PULUMI_MID_MEMPROFILE_PATH", "")
	if err != nil {
		ts.Logger.Error("error getting memprofile path", slog.Any("error", err))
	}
	if memprofilePath == "" {
		ts.Logger.Debug("memory profiling disabled")
	} else {
		ts.Logger = ts.Logger.With(slog.String("memprofile_path", memprofilePath))
		ts.Logger.Info("memory profiling enabled")
		memprofileFile, err := os.Create(memprofilePath)
		if err != nil {
			ts.Logger.Error(
				"error opening memprofile path",
				slog.String("memprofile_path", memprofilePath),
				slog.Any("error", err),
			)
			goto otel
		}
		ts.MemprofileFile = memprofileFile
	}

otel:
	otlpEndpoint, err := env.GetDefault("PULUMI_MID_OTLP_ENDPOINT", "")
	if err != nil {
		ts.Logger.Error("error getting otel OTLP endpoint", slog.Any("error", err))
	}
	if otlpEndpoint == "" {
		ts.Logger.Debug("telemetry disabled")
	} else {
		ts.Logger = ts.Logger.With(slog.String("otlp_endpoint", otlpEndpoint))
		ts.Logger.Info("telemetry enabled")
		exporter, err := otlptrace.New(
			ts.Context,
			otlptracegrpc.NewClient(
				otlptracegrpc.WithInsecure(),
				otlptracegrpc.WithEndpoint(otlpEndpoint),
			),
		)
		if err != nil {
			ts.Logger.Error("error setting up otlptrace", slog.Any("error", err))
			goto end
		}

		res, err := resource.New(
			context.Background(),
			resource.WithAttributes(
				attribute.String("service.name", "pulumi-resource-mid"),
				attribute.String("library.language", "go"),
			),
		)
		if err != nil {
			ts.Logger.Error("error setting up otel resource", slog.Any("error", err))
			goto end
		}

		otel.SetTracerProvider(
			sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(res),
			),
		)
	}

end:
	ts.Logger.Debug("started telemetry")
	return ts
}

// OtelJSON JSON marshals any value into an otel attribute.KeyValue
func OtelJSON(key string, value any) attribute.KeyValue {
	data, err := json.Marshal(value)
	if err != nil {
		return attribute.String(key, "err!"+err.Error())
	}
	return attribute.String(key, string(data))
}
