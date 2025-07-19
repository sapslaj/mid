package agent

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

var Tracer = otel.Tracer("mid/provider/executor")

func CallAgent[I any, O any](
	ctx context.Context,
	resourceConnection *midtypes.Connection,
	resourceConfig *midtypes.ResourceConfig,
	rpcFunction rpc.RPCFunction,
	args I,
) (O, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/agent.CallAgent", trace.WithAttributes(
		attribute.String("rpc.function", string(rpcFunction)),
		telemetry.OtelJSON("rpc.args", args),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, resourceConnection)
	config := midtypes.GetResourceConfig(ctx, resourceConfig)

	call := rpc.RPCCall[I]{
		RPCFunction: rpcFunction,
		Args:        args,
	}
	var output O

	callResult, err := executor.CallAgent[I, O](ctx, connection, config, call)
	output = callResult.Result
	span.SetAttributes(telemetry.OtelJSON("rpc.result", output))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return output, err
	}

	if callResult.Error != "" {
		span.SetAttributes(attribute.String("rpc.error", callResult.Error))
		err = errors.New(callResult.Error)
		span.SetStatus(codes.Error, err.Error())
		return output, err
	}

	span.SetStatus(codes.Ok, "")
	return output, nil
}

func ToOptional[T comparable](v T) *T {
	var zero T
	if zero == v {
		return nil
	}
	return &v
}
