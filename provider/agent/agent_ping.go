package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
)

type AgentPing struct{}

type AgentPingInput struct {
	Ping string `pulumi:"ping,optional"`
}

type AgentPingOutput struct {
	Ping string `pulumi:"ping"`
	Pong string `pulumi:"pong"`
}

func (f AgentPing) Invoke(
	ctx context.Context,
	req infer.FunctionRequest[AgentPingInput],
) (infer.FunctionResponse[AgentPingOutput], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/agent/agentPing.Call", trace.WithAttributes(
		attribute.String("pulumi.function", "mid:agent:agentPing"),
		telemetry.OtelJSON("pulumi.input", req.Input),
	))
	defer span.End()

	out, err := CallAgent[rpc.AgentPingArgs, rpc.AgentPingResult](ctx, rpc.RPCAgentPing, rpc.AgentPingArgs{
		Ping: req.Input.Ping,
	})
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	output := AgentPingOutput{
		Ping: out.Ping,
		Pong: out.Pong,
	}
	span.SetAttributes(telemetry.OtelJSON("pulumi.output", output))

	return infer.FunctionResponse[AgentPingOutput]{
		Output: output,
	}, err
}
