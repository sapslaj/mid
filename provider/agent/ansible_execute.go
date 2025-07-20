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

type AnsibleExecute struct{}

type AnsibleExecuteInput struct {
	Name               string                   `pulumi:"name"`
	Args               map[string]any           `pulumi:"args"`
	Environment        map[string]string        `pulumi:"environment,optional"`
	Check              bool                     `pulumi:"check,optional"`
	DebugKeepTempFiles bool                     `pulumi:"debugKeepTempFiles,optional"`
	Connection         *midtypes.Connection     `pulumi:"connection,optional"`
	Config             *midtypes.ResourceConfig `pulumi:"config,optional"`
}

type AnsibleExecuteOutput struct {
	AnsibleExecuteInput
	Stderr       string         `pulumi:"stderr"`
	Stdout       string         `pulumi:"stdout"`
	ExitCode     int            `pulumi:"exitCode"`
	Result       map[string]any `pulumi:"result"`
	DebugTempDir *string        `pulumi:"debugTempDir,optional"`
}

func (f AnsibleExecute) Invoke(
	ctx context.Context,
	req infer.FunctionRequest[AnsibleExecuteInput],
) (infer.FunctionResponse[AnsibleExecuteOutput], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/agent/ansibleExecute.Call", trace.WithAttributes(
		attribute.String("pulumi.function", "mid:agent:ansibleExecute"),
		telemetry.OtelJSON("pulumi.input", req.Input),
	))
	defer span.End()

	out, err := CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](
		ctx,
		req.Input.Connection,
		req.Input.Config,
		rpc.RPCAnsibleExecute,
		rpc.AnsibleExecuteArgs{
			Name:               req.Input.Name,
			Args:               req.Input.Args,
			Environment:        req.Input.Environment,
			Check:              req.Input.Check,
			DebugKeepTempFiles: req.Input.DebugKeepTempFiles,
		},
	)

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	output := AnsibleExecuteOutput{
		AnsibleExecuteInput: req.Input,
		Stderr:              string(out.Stderr),
		Stdout:              string(out.Stdout),
		ExitCode:            out.ExitCode,
		Result:              out.Result,
		DebugTempDir:        ToOptional(out.DebugTempDir),
	}
	span.SetAttributes(telemetry.OtelJSON("pulumi.output", output))

	return infer.FunctionResponse[AnsibleExecuteOutput]{
		Output: output,
	}, err
}
