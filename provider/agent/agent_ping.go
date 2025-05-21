package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
)

type AgentPing struct{}

type AgentPingInputs struct {
	Ping string `pulumi:"ping,optional"`
}

type AgentPingOutputs struct {
	Ping string `pulumi:"ping"`
	Pong string `pulumi:"pong"`
}

func (f AgentPing) Call(ctx context.Context, input AgentPingInputs) (AgentPingOutputs, error) {
	ctx, span := Tracer.Start(ctx, "mid:agent:agentPing.Call", trace.WithAttributes(
		attribute.String("pulumi.function.token", "mid:agent:agentPing"),
		telemetry.OtelJSON("pulumi.function.inputs", input),
	))
	defer span.End()

	out, err := CallAgent[rpc.AgentPingArgs, rpc.AgentPingResult](ctx, rpc.RPCAgentPing, rpc.AgentPingArgs{
		Ping: input.Ping,
	})
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	outputs := AgentPingOutputs{
		Ping: out.Ping,
		Pong: out.Pong,
	}
	span.SetAttributes(telemetry.OtelJSON("pulumi.function.outputs", outputs))

	return outputs, err
}
