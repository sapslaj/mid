package resource

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/sapslaj/mid/pkg/pdiff"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type Exec struct{}

type ExecArgs struct {
	Create              midtypes.ExecCommand     `pulumi:"create"`
	Update              *midtypes.ExecCommand    `pulumi:"update,optional"`
	Delete              *midtypes.ExecCommand    `pulumi:"delete,optional"`
	ExpandArgumentVars  *bool                    `pulumi:"expandArgumentVars,optional"`
	DeleteBeforeReplace *bool                    `pulumi:"deleteBeforeReplace,optional"`
	Dir                 *string                  `pulumi:"dir,optional"`
	Environment         *map[string]string       `pulumi:"environment,optional"`
	Logging             *midtypes.ExecLogging    `pulumi:"logging,optional"`
	Connection          *midtypes.Connection     `pulumi:"connection,optional"`
	Config              *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers            *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type ExecState struct {
	ExecArgs
	Stdout   string                  `pulumi:"stdout"`
	Stderr   string                  `pulumi:"stderr"`
	Triggers midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r Exec) argsToRPCCall(input ExecArgs, lifecycle string) (rpc.RPCCall[rpc.ExecArgs], error) {
	environment := map[string]string{}

	var execCommand midtypes.ExecCommand
	switch lifecycle {
	case "create":
		execCommand = input.Create
	case "update":
		if input.Update != nil {
			execCommand = *input.Update
		} else {
			execCommand = input.Create
		}
	case "delete":
		if input.Delete == nil {
			return rpc.RPCCall[rpc.ExecArgs]{}, nil
		}
		execCommand = *input.Delete
	default:
		panic("unknown lifecycle: " + lifecycle)
	}

	chdir := ""
	if input.Dir != nil {
		chdir = *input.Dir
	}
	if execCommand.Dir != nil {
		chdir = *execCommand.Dir
	}

	if input.Environment != nil {
		maps.Copy(environment, *input.Environment)
	}
	if execCommand.Environment != nil {
		maps.Copy(environment, *execCommand.Environment)
	}

	stdin := []byte{}
	if execCommand.Stdin != nil {
		stdin = []byte(*execCommand.Stdin)
	}

	return rpc.RPCCall[rpc.ExecArgs]{
		RPCFunction: rpc.RPCExec,
		Args: rpc.ExecArgs{
			Command:            execCommand.Command,
			Dir:                chdir,
			Environment:        environment,
			Stdin:              stdin,
			ExpandArgumentVars: input.ExpandArgumentVars != nil && *input.ExpandArgumentVars,
		},
	}, nil
}

func (r Exec) updateState(inputs ExecArgs, state ExecState, changed bool) ExecState {
	state.ExecArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r Exec) updateStateFromRPCResult(
	inputs ExecArgs,
	state ExecState,
	result rpc.RPCResult[rpc.ExecResult],
) ExecState {
	logging := midtypes.ExecLoggingStdoutAndStderr
	if inputs.Logging != nil {
		logging = *inputs.Logging
	}
	switch logging {
	case midtypes.ExecLoggingNone:
		state.Stderr = ""
		state.Stdout = ""
	case midtypes.ExecLoggingStderr:
		state.Stderr = string(result.Result.Stderr)
		state.Stdout = ""
	case midtypes.ExecLoggingStdout:
		state.Stderr = ""
		state.Stdout = string(result.Result.Stdout)
	case midtypes.ExecLoggingStdoutAndStderr:
		state.Stderr = string(result.Result.Stderr)
		state.Stdout = string(result.Result.Stdout)
	default:
		panic("unknown logging: " + logging)
	}
	return state
}

func (r Exec) runRPCExec(
	ctx context.Context,
	connection midtypes.Connection,
	config midtypes.ResourceConfig,
	inputs ExecArgs,
	state ExecState,
	lifecycle string,
) (ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.runRPCExec", trace.WithAttributes(
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("inputs", inputs),
		attribute.String("lifecycle", lifecycle),
	))
	defer span.End()

	if connection.Host != nil {
		span.SetAttributes(
			attribute.String("connection.host", *connection.Host),
		)
	}

	call, err := r.argsToRPCCall(inputs, lifecycle)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	result, err := executor.CallAgent[rpc.ExecArgs, rpc.ExecResult](ctx, connection, config, call)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if result.Error != "" {
		err = fmt.Errorf(
			"mid encountered an issue running command '%v': %s",
			call.Args.Command,
			result.Error,
		)
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if result.Result.ExitCode != 0 {
		err = fmt.Errorf(
			"command '%v' exited with status %d: stderr=%s stdout=%s",
			call.Args.Command,
			result.Result.ExitCode,
			result.Result.Stderr,
			result.Result.Stdout,
		)
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state = r.updateStateFromRPCResult(inputs, state, result)
	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Exec) Diff(ctx context.Context, req infer.DiffRequest[ExecArgs, ExecState]) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:Exec"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	diff := p.DiffResponse{
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	if req.Inputs.DeleteBeforeReplace != nil {
		diff.DeleteBeforeReplace = *req.Inputs.DeleteBeforeReplace
	}

	diff = pdiff.MergeDiffResponses(
		diff,
		pdiff.DiffAllAttributesExcept(req.Inputs, req.State, []string{
			"deleteBeforeReplace",
			"connection",
			"config",
			"triggers",
		}),
		midtypes.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r Exec) Create(
	ctx context.Context,
	req infer.CreateRequest[ExecArgs],
) (infer.CreateResponse[ExecState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:Exec"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(req.Inputs, ExecState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if req.DryRun {
		return infer.CreateResponse[ExecState]{
			ID:     req.Name,
			Output: state,
		}, nil
	}

	state, err := r.runRPCExec(ctx, connection, config, req.Inputs, state, "create")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[ExecState]{
			ID:     req.Name,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[ExecState]{
		ID:     req.Name,
		Output: state,
	}, nil
}

func (r Exec) Update(
	ctx context.Context,
	req infer.UpdateRequest[ExecArgs, ExecState],
) (infer.UpdateResponse[ExecState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:Exec"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(req.Inputs, req.State, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if req.DryRun {
		return infer.UpdateResponse[ExecState]{
			Output: state,
		}, nil
	}

	var err error
	state, err = r.runRPCExec(ctx, connection, config, req.Inputs, state, "update")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[ExecState]{
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[ExecState]{
		Output: state,
	}, nil
}

func (r Exec) Delete(ctx context.Context, req infer.DeleteRequest[ExecState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:Exec"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	if req.State.Delete == nil {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	connection := midtypes.GetConnection(ctx, req.State.Connection)
	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	_, err := r.runRPCExec(ctx, connection, config, req.State.ExecArgs, req.State, "delete")
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && config.GetDeleteUnreachable() {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetAttributes(attribute.Bool("unreachable.deleted", true))
			span.SetStatus(codes.Ok, "")
			return infer.DeleteResponse{}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.DeleteResponse{}, nil
}
