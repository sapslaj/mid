package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
)

type Exec struct{}

type ExecInputs struct {
	Command     []string          `pulumi:"command"`
	Dir         string            `pulumi:"dir,optional"`
	Environment map[string]string `pulumi:"environment,optional"`
	Stdin       string            `pulumi:"stdin,optional"`
}

type ExecOutputs struct {
	ExecInputs
	Stdout   string `pulumi:"stdout"`
	Stderr   string `pulumi:"stderr"`
	ExitCode int    `pulumi:"exitCode"`
	Pid      int    `pulumi:"pid"`
}

func (f Exec) Call(ctx context.Context, input ExecInputs) (ExecOutputs, error) {
	ctx, span := Tracer.Start(ctx, "mid:agent:exec.Call", trace.WithAttributes(
		attribute.String("pulumi.function.token", "mid:agent:exec"),
		telemetry.OtelJSON("pulumi.function.inputs", input),
	))
	defer span.End()

	out, err := CallAgent[rpc.ExecArgs, rpc.ExecResult](ctx, rpc.RPCExec, rpc.ExecArgs{
		Command:     input.Command,
		Dir:         input.Dir,
		Environment: input.Environment,
		Stdin:       []byte(input.Stdin),
	})

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	outputs := ExecOutputs{
		ExecInputs: input,
		Stdout:     string(out.Stdout),
		Stderr:     string(out.Stderr),
		ExitCode:   out.ExitCode,
		Pid:        out.Pid,
	}
	span.SetAttributes(telemetry.OtelJSON("pulumi.function.outputs", outputs))

	return outputs, err
}
