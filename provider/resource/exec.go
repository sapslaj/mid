package resource

import (
	"context"
	"errors"
	"fmt"
	"maps"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
)

type Exec struct{}

type ExecArgs struct {
	Create              types.ExecCommand    `pulumi:"create"`
	Update              *types.ExecCommand   `pulumi:"update,optional"`
	Delete              *types.ExecCommand   `pulumi:"delete,optional"`
	ExpandArgumentVars  *bool                `pulumi:"expandArgumentVars,optional"`
	DeleteBeforeReplace *bool                `pulumi:"deleteBeforeReplace,optional"`
	Dir                 *string              `pulumi:"dir,optional"`
	Environment         *map[string]string   `pulumi:"environment,optional"`
	Logging             *types.ExecLogging   `pulumi:"logging,optional"`
	Triggers            *types.TriggersInput `pulumi:"triggers,optional"`
}

type ExecState struct {
	ExecArgs
	Stdout   string               `pulumi:"stdout"`
	Stderr   string               `pulumi:"stderr"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

func (r Exec) canUseRPC(input ExecArgs) bool {
	if input.ExpandArgumentVars != nil {
		if *input.ExpandArgumentVars {
			return false
		}
	}
	return true
}

func (r Exec) argsToRPCCall(input ExecArgs, lifecycle string) (rpc.RPCCall[rpc.ExecArgs], error) {
	environment := map[string]string{}

	var execCommand types.ExecCommand
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
			Command:     execCommand.Command,
			Dir:         chdir,
			Environment: environment,
			Stdin:       stdin,
		},
	}, nil
}

func (r Exec) argsToTaskParameters(
	input ExecArgs,
	lifecycle string,
) (ansible.CommandParameters, map[string]string, error) {
	environment := map[string]string{}

	var execCommand types.ExecCommand
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
			return ansible.CommandParameters{}, environment, nil
		}
		execCommand = *input.Delete
	default:
		panic("unknown lifecycle: " + lifecycle)
	}

	chdir := input.Dir
	if execCommand.Dir != nil {
		chdir = execCommand.Dir
	}
	expandArgumentVars := false
	if input.ExpandArgumentVars != nil {
		expandArgumentVars = *input.ExpandArgumentVars
	}

	if input.Environment != nil {
		maps.Copy(environment, *input.Environment)
	}
	if execCommand.Environment != nil {
		maps.Copy(environment, *execCommand.Environment)
	}

	return ansible.CommandParameters{
		Argv:               ptr.Of(execCommand.Command),
		Chdir:              chdir,
		Stdin:              execCommand.Stdin,
		ExpandArgumentVars: ptr.Of(expandArgumentVars),
		StripEmptyEnds:     ptr.Of(false),
	}, environment, nil
}

func (r Exec) updateState(inputs ExecArgs, state ExecState, changed bool) ExecState {
	state.ExecArgs = inputs
	state.Triggers = types.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r Exec) updateStateFromRPCResult(
	inputs ExecArgs,
	state ExecState,
	result rpc.RPCResult[rpc.ExecResult],
) ExecState {
	logging := types.ExecLoggingStdoutAndStderr
	if inputs.Logging != nil {
		logging = *inputs.Logging
	}
	switch logging {
	case types.ExecLoggingNone:
		state.Stderr = ""
		state.Stdout = ""
	case types.ExecLoggingStderr:
		state.Stderr = string(result.Result.Stderr)
		state.Stdout = ""
	case types.ExecLoggingStdout:
		state.Stderr = ""
		state.Stdout = string(result.Result.Stdout)
	case types.ExecLoggingStdoutAndStderr:
		state.Stderr = string(result.Result.Stderr)
		state.Stdout = string(result.Result.Stdout)
	default:
		panic("unknown logging: " + logging)
	}
	return state
}

func (r Exec) updateStateFromOutput(news ExecArgs, olds ExecState, output ansible.CommandReturn) ExecState {
	logging := types.ExecLoggingStdoutAndStderr
	if news.Logging != nil {
		logging = *news.Logging
	}
	switch logging {
	case types.ExecLoggingNone:
		olds.Stderr = ""
		olds.Stdout = ""
	case types.ExecLoggingStderr:
		if output.Stderr != nil {
			olds.Stderr = *output.Stderr
		}
		olds.Stdout = ""
	case types.ExecLoggingStdout:
		olds.Stderr = ""
		if output.Stdout != nil {
			olds.Stdout = *output.Stdout
		}
	case types.ExecLoggingStdoutAndStderr:
		if output.Stderr != nil {
			olds.Stderr = *output.Stderr
		}
		if output.Stdout != nil {
			olds.Stdout = *output.Stdout
		}
	default:
		panic("unknown logging: " + logging)
	}
	return olds
}

func (r Exec) runRPCExec(
	ctx context.Context,
	connection *types.Connection,
	inputs ExecArgs,
	state ExecState,
	lifecycle string,
) (ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.runRPCExec", trace.WithAttributes(
		attribute.String("connection.host", *connection.Host),
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("inputs", inputs),
		attribute.String("lifecycle", lifecycle),
	))
	defer span.End()

	call, err := r.argsToRPCCall(inputs, lifecycle)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	result, err := executor.CallAgent[rpc.ExecArgs, rpc.ExecResult](ctx, connection, call)
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

func (r Exec) runRPCAnsibleExecute(
	ctx context.Context,
	connection *types.Connection,
	inputs ExecArgs,
	state ExecState,
	lifecycle string,
) (ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Exec.runRPCAnsibleExecute", trace.WithAttributes(
		attribute.String("connection.host", *connection.Host),
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("inputs", inputs),
		attribute.String("lifecycle", lifecycle),
	))
	defer span.End()

	parameters, environment, err := r.argsToTaskParameters(inputs, lifecycle)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}
	call.Args.Environment = environment

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, connection, call)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	result, err := ansible.CommandReturnFromRPCResult(callResult)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if callResult.Error != "" {
		err = fmt.Errorf(
			"mid encountered an issue running command '%v': %s",
			parameters.Argv,
			callResult.Error,
		)
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if !callResult.Result.Success {
		if result.Rc == nil {
			err = fmt.Errorf(
				"mid encountered an issue running command '%v': stderr=%s stdout=%s",
				parameters.Argv,
				callResult.Result.Stderr,
				callResult.Result.Stdout,
			)
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		stdout := "<nil>"
		if result.Stdout != nil {
			stdout = *result.Stdout
		}
		stderr := "<nil>"
		if result.Stderr != nil {
			stderr = *result.Stderr
		}

		err = fmt.Errorf(
			"command '%v' exited with status %d: stderr=%s stdout=%s",
			parameters.Argv,
			*result.Rc,
			stderr,
			stdout,
		)
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state = r.updateStateFromOutput(inputs, state, result)
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
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	if req.Inputs.DeleteBeforeReplace != nil {
		diff.DeleteBeforeReplace = *req.Inputs.DeleteBeforeReplace
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(req.State, req.Inputs, []string{
			"create",
			"update",
			"delete",
			"expandArgumentVars",
			"dir",
			"environment",
			"logging",
		}),
		types.DiffTriggers(req.State, req.Inputs),
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

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(req.Inputs, ExecState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[ExecState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	if r.canUseRPC(req.Inputs) {
		span.SetAttributes(attribute.String("exec.strategy", "rpc"))
	} else {
		span.SetAttributes(attribute.String("exec.strategy", "ansible"))
	}

	if req.DryRun {
		return infer.CreateResponse[ExecState]{
			ID:     id,
			Output: state,
		}, nil
	}

	if r.canUseRPC(req.Inputs) {
		state, err = r.runRPCExec(ctx, config.Connection, req.Inputs, state, "create")
	} else {
		state, err = r.runRPCAnsibleExecute(ctx, config.Connection, req.Inputs, state, "create")
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[ExecState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[ExecState]{
		ID:     id,
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

	config := infer.GetConfig[types.Config](ctx)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if r.canUseRPC(req.Inputs) {
		span.SetAttributes(attribute.String("exec.strategy", "rpc"))
	} else {
		span.SetAttributes(attribute.String("exec.strategy", "ansible"))
	}

	if req.DryRun {
		return infer.UpdateResponse[ExecState]{
			Output: state,
		}, nil
	}

	var err error
	if r.canUseRPC(req.Inputs) {
		state, err = r.runRPCExec(ctx, config.Connection, req.Inputs, state, "update")
	} else {
		state, err = r.runRPCAnsibleExecute(ctx, config.Connection, req.Inputs, state, "update")
	}
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

	config := infer.GetConfig[types.Config](ctx)

	if r.canUseRPC(req.State.ExecArgs) {
		span.SetAttributes(attribute.String("exec.strategy", "rpc"))
	} else {
		span.SetAttributes(attribute.String("exec.strategy", "ansible"))
	}

	var err error
	if r.canUseRPC(req.State.ExecArgs) {
		_, err = r.runRPCExec(ctx, config.Connection, req.State.ExecArgs, req.State, "delete")
	} else {
		_, err = r.runRPCAnsibleExecute(ctx, config.Connection, req.State.ExecArgs, req.State, "delete")
	}
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
