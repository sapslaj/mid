package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
)

type AnsibleExecute struct{}

type AnsibleExecuteInputs struct {
	Name               string            `pulumi:"name"`
	Args               map[string]any    `pulumi:"args"`
	Environment        map[string]string `pulumi:"environment,optional"`
	Check              bool              `pulumi:"check,optional"`
	DebugKeepTempFiles bool              `pulumi:"debugKeepTempFiles,optional"`
}

type AnsibleExecuteOutputs struct {
	AnsibleExecuteInputs
	Stderr       string         `pulumi:"stderr"`
	Stdout       string         `pulumi:"stdout"`
	ExitCode     int            `pulumi:"exitCode"`
	Result       map[string]any `pulumi:"result"`
	DebugTempDir *string        `pulumi:"debugTempDir,optional"`
}

func (f AnsibleExecute) Call(ctx context.Context, input AnsibleExecuteInputs) (AnsibleExecuteOutputs, error) {
	ctx, span := Tracer.Start(ctx, "mid:agent:ansibleExecute.Call", trace.WithAttributes(
		attribute.String("pulumi.function.token", "mid:agent:ansibleExecute"),
		telemetry.OtelJSON("pulumi.function.inputs", input),
	))
	defer span.End()

	out, err := CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](
		ctx,
		rpc.RPCAnsibleExecute,
		rpc.AnsibleExecuteArgs{
			Name:               input.Name,
			Args:               input.Args,
			Environment:        input.Environment,
			Check:              input.Check,
			DebugKeepTempFiles: input.DebugKeepTempFiles,
		},
	)

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	outputs := AnsibleExecuteOutputs{
		AnsibleExecuteInputs: input,
		Stderr:               string(out.Stderr),
		Stdout:               string(out.Stdout),
		ExitCode:             out.ExitCode,
		Result:               out.Result,
		DebugTempDir:         ToOptional(out.DebugTempDir),
	}
	span.SetAttributes(telemetry.OtelJSON("pulumi.function.outputs", outputs))

	return outputs, err
}
