package resource

import (
	"context"
	"errors"
	"fmt"
	"maps"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"
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

func (r Exec) argsToTaskParameters(input ExecArgs, lifecycle string) (ansible.CommandParameters, map[string]string, error) {
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

func (r Exec) updateState(olds ExecState, news ExecArgs, changed bool) ExecState {
	olds.ExecArgs = news
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Exec) updateStateFromRPCResult(olds ExecState, news ExecArgs, result rpc.RPCResult[rpc.ExecResult]) ExecState {
	logging := types.ExecLoggingStdoutAndStderr
	if news.Logging != nil {
		logging = *news.Logging
	}
	switch logging {
	case types.ExecLoggingNone:
		olds.Stderr = ""
		olds.Stdout = ""
	case types.ExecLoggingStderr:
		olds.Stderr = string(result.Result.Stderr)
		olds.Stdout = ""
	case types.ExecLoggingStdout:
		olds.Stderr = ""
		olds.Stdout = string(result.Result.Stdout)
	case types.ExecLoggingStdoutAndStderr:
		olds.Stderr = string(result.Result.Stderr)
		olds.Stdout = string(result.Result.Stdout)
	default:
		panic("unknown logging: " + logging)
	}
	return olds
}

func (r Exec) updateStateFromOutput(olds ExecState, news ExecArgs, output ansible.CommandReturn) ExecState {
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
	state ExecState,
	input ExecArgs,
	lifecycle string,
) (ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Exec.runRPCExec", trace.WithAttributes(
		attribute.String("connection.host", *connection.Host),
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("input", input),
		attribute.String("lifecycle", lifecycle),
	))
	defer span.End()

	call, err := r.argsToRPCCall(input, lifecycle)
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

	state = r.updateStateFromRPCResult(state, input, result)
	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Exec) runRPCAnsibleExecute(
	ctx context.Context,
	connection *types.Connection,
	state ExecState,
	input ExecArgs,
	lifecycle string,
) (ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Exec.runRPCAnsibleExecute", trace.WithAttributes(
		attribute.String("connection.host", *connection.Host),
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("input", input),
		attribute.String("lifecycle", lifecycle),
	))
	defer span.End()

	parameters, environment, err := r.argsToTaskParameters(input, lifecycle)
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

	state = r.updateStateFromOutput(state, input, result)
	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Exec) Diff(
	ctx context.Context,
	id string,
	olds ExecState,
	news ExecArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Exec.Diff", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
	))
	defer span.End()

	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	if news.DeleteBeforeReplace != nil {
		diff.DeleteBeforeReplace = *news.DeleteBeforeReplace
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
			"create",
			"update",
			"delete",
			"expandArgumentVars",
			"dir",
			"environment",
			"logging",
		}),
		types.DiffTriggers(olds, news),
	)

	span.SetStatus(codes.Ok, "")
	return diff, nil
}

func (r Exec) Create(
	ctx context.Context,
	name string,
	input ExecArgs,
	preview bool,
) (string, ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Exec.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(ExecState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", state, err
	}
	span.SetAttributes(attribute.String("id", id))

	if r.canUseRPC(input) {
		span.SetAttributes(attribute.String("exec.strategy", "rpc"))
	} else {
		span.SetAttributes(attribute.String("exec.strategy", "ansible"))
	}

	if preview {
		return id, state, nil
	}

	if r.canUseRPC(input) {
		state, err = r.runRPCExec(ctx, config.Connection, state, input, "create")
	} else {
		state, err = r.runRPCAnsibleExecute(ctx, config.Connection, state, input, "create")
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, state, nil
}

func (r Exec) Update(
	ctx context.Context,
	id string,
	olds ExecState,
	news ExecArgs,
	preview bool,
) (ExecState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Exec.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if r.canUseRPC(news) {
		span.SetAttributes(attribute.String("exec.strategy", "rpc"))
	} else {
		span.SetAttributes(attribute.String("exec.strategy", "ansible"))
	}

	if preview {
		return olds, nil
	}

	var err error
	if r.canUseRPC(news) {
		olds, err = r.runRPCExec(ctx, config.Connection, olds, news, "update")
	} else {
		olds, err = r.runRPCAnsibleExecute(ctx, config.Connection, olds, news, "update")
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	span.SetStatus(codes.Ok, "")
	return olds, nil
}

func (r Exec) Delete(ctx context.Context, id string, props ExecState) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:Exec.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	if props.Delete == nil {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	if r.canUseRPC(props.ExecArgs) {
		span.SetAttributes(attribute.String("exec.strategy", "rpc"))
	} else {
		span.SetAttributes(attribute.String("exec.strategy", "ansible"))
	}

	var err error
	if r.canUseRPC(props.ExecArgs) {
		_, err = r.runRPCExec(ctx, config.Connection, props, props.ExecArgs, "delete")
	} else {
		_, err = r.runRPCAnsibleExecute(ctx, config.Connection, props, props.ExecArgs, "delete")
	}
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && config.GetDeleteUnreachable() {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetAttributes(attribute.Bool("unreachable.deleted", true))
			span.SetStatus(codes.Ok, "")
			return nil
		}
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
