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

type Exec struct{}

type ExecInput struct {
	Command     []string          `pulumi:"command"`
	Dir         string            `pulumi:"dir,optional"`
	Environment map[string]string `pulumi:"environment,optional"`
	Stdin       string            `pulumi:"stdin,optional"`
}

type ExecOutput struct {
	ExecInput
	Stdout   string `pulumi:"stdout"`
	Stderr   string `pulumi:"stderr"`
	ExitCode int    `pulumi:"exitCode"`
	Pid      int    `pulumi:"pid"`
}

func (f Exec) Invoke(
	ctx context.Context,
	req infer.FunctionRequest[ExecInput],
) (infer.FunctionResponse[ExecOutput], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/agent/exec.Call", trace.WithAttributes(
		attribute.String("pulumi.function", "mid:agent:exec"),
		telemetry.OtelJSON("pulumi.input", req.Input),
	))
	defer span.End()

	out, err := CallAgent[rpc.ExecArgs, rpc.ExecResult](ctx, rpc.RPCExec, rpc.ExecArgs{
		Command:     req.Input.Command,
		Dir:         req.Input.Dir,
		Environment: req.Input.Environment,
		Stdin:       []byte(req.Input.Stdin),
	})

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	output := ExecOutput{
		ExecInput: req.Input,
		Stdout:    string(out.Stdout),
		Stderr:    string(out.Stderr),
		ExitCode:  out.ExitCode,
		Pid:       out.Pid,
	}
	span.SetAttributes(telemetry.OtelJSON("pulumi.outputs", output))

	return infer.FunctionResponse[ExecOutput]{
		Output: output,
	}, err
}
