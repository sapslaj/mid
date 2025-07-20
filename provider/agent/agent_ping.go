package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/midtypes"
)

type AgentPing struct{}

type AgentPingInput struct {
	Ping       string                   `pulumi:"ping,optional"`
	Connection *midtypes.Connection     `pulumi:"connection,optional"`
	Config     *midtypes.ResourceConfig `pulumi:"config,optional"`
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

	out, err := CallAgent[rpc.AgentPingArgs, rpc.AgentPingResult](
		ctx,
		req.Input.Connection,
		req.Input.Config,
		rpc.RPCAgentPing,
		rpc.AgentPingArgs{
			Ping: req.Input.Ping,
		},
	)
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
