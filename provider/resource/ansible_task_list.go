package resource

import (
	"context"
	"errors"
	"fmt"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/pdiff"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type AnsibleTaskList struct{}

type AnsibleTaskListArgsTask struct {
	Module       string             `pulumi:"module"`
	Args         map[string]any     `pulumi:"args"`
	Environment  *map[string]string `pulumi:"environment,optional"`
	Check        *bool              `pulumi:"check,optional"`
	IgnoreErrors *bool              `pulumi:"ignoreErrors,optional"`
}

type AnsibleTaskListArgsTasks struct {
	Create []AnsibleTaskListArgsTask  `pulumi:"create"`
	Update *[]AnsibleTaskListArgsTask `pulumi:"update,optional"`
	Delete *[]AnsibleTaskListArgsTask `pulumi:"delete,optional"`
}

type AnsibleTaskListArgs struct {
	Tasks      AnsibleTaskListArgsTasks `pulumi:"tasks"`
	Connection *midtypes.Connection     `pulumi:"connection,optional"`
	Config     *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers   *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type AnsibleTaskListStateTaskResult struct {
	AnsibleTaskListArgsTask
	Stderr   string         `pulumi:"stderr"`
	Stdout   string         `pulumi:"stdout"`
	ExitCode int            `pulumi:"exitCode"`
	Success  bool           `pulumi:"success"`
	Result   map[string]any `pulumi:"result"`
}

type AnsibleTaskListStateResults struct {
	Lifecycle string                           `pulumi:"lifecycle"`
	Tasks     []AnsibleTaskListStateTaskResult `pulumi:"tasks"`
}

type AnsibleTaskListState struct {
	AnsibleTaskListArgs
	Results  AnsibleTaskListStateResults `pulumi:"results"`
	Triggers midtypes.TriggersOutput     `pulumi:"triggers"`
}

func (r AnsibleTaskList) updateState(
	inputs AnsibleTaskListArgs,
	state AnsibleTaskListState,
	changed bool,
) AnsibleTaskListState {
	state.AnsibleTaskListArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r AnsibleTaskList) Diff(
	ctx context.Context,
	req infer.DiffRequest[AnsibleTaskListArgs, AnsibleTaskListState],
) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/AnsibleTaskList.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:AnsibleTaskList"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	diff := p.DiffResponse{
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: true,
	}

	diff = pdiff.MergeDiffResponses(
		diff,
		pdiff.DiffAllAttributesExcept(req.Inputs, req.State, []string{
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

func (r AnsibleTaskList) run(
	ctx context.Context,
	inputs AnsibleTaskListArgs,
	state AnsibleTaskListState,
	lifecycle string,
) (AnsibleTaskListState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/AnsibleTaskList.run", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	state.Results.Lifecycle = lifecycle

	var taskList []AnsibleTaskListArgsTask
	switch lifecycle {
	case "create":
		taskList = inputs.Tasks.Create
	case "update":
		if inputs.Tasks.Update == nil {
			taskList = inputs.Tasks.Create
		} else {
			taskList = *inputs.Tasks.Update
		}
	case "delete":
		if inputs.Tasks.Delete == nil {
			return state, nil
		}
		taskList = *inputs.Tasks.Delete
	}

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	canConnect, err := executor.CanConnect(ctx, connection, config, 10)
	if !canConnect && err == nil {
		err = executor.ErrUnreachable
	}
	if err != nil {
		return state, err
	}

	state.Results.Tasks = []AnsibleTaskListStateTaskResult{}

	for _, task := range taskList {
		ignoreErrors := false
		if task.IgnoreErrors != nil {
			ignoreErrors = *task.IgnoreErrors
		}

		call := rpc.RPCCall[rpc.AnsibleExecuteArgs]{
			RPCFunction: rpc.RPCAnsibleExecute,
			Args: rpc.AnsibleExecuteArgs{
				Name: task.Module,
				Args: task.Args,
			},
		}
		if task.Environment != nil {
			call.Args.Environment = *task.Environment
		}
		if task.Check != nil {
			call.Args.Check = *task.Check
		}

		callResult, err := executor.CallAgent[
			rpc.AnsibleExecuteArgs,
			rpc.AnsibleExecuteResult,
		](ctx, connection, config, call)
		if err != nil && !ignoreErrors {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		state.Results.Tasks = append(state.Results.Tasks, AnsibleTaskListStateTaskResult{
			AnsibleTaskListArgsTask: task,
			Stderr:                  string(callResult.Result.Stderr),
			Stdout:                  string(callResult.Result.Stdout),
			ExitCode:                callResult.Result.ExitCode,
			Success:                 callResult.Result.Success,
			Result:                  callResult.Result.Result,
		})

		if callResult.Error != "" && !ignoreErrors {
			err = errors.New(callResult.Error)
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		if !callResult.Result.Success && !ignoreErrors {
			err = fmt.Errorf(
				"error running Ansible task: exitcode=%d stderr=%s stdout=%s",
				callResult.Result.ExitCode,
				string(callResult.Result.Stderr),
				string(callResult.Result.Stdout),
			)
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
	}

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r AnsibleTaskList) Create(
	ctx context.Context,
	req infer.CreateRequest[AnsibleTaskListArgs],
) (infer.CreateResponse[AnsibleTaskListState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/AnsibleTaskList.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:AnsibleTaskList"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	state := r.updateState(req.Inputs, AnsibleTaskListState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if req.DryRun {
		span.SetStatus(codes.Ok, "")
		return infer.CreateResponse[AnsibleTaskListState]{
			ID:     req.Name,
			Output: state,
		}, nil
	}

	var err error
	state, err = r.run(ctx, req.Inputs, state, "create")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[AnsibleTaskListState]{
			ID:     req.Name,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[AnsibleTaskListState]{
		ID:     req.Name,
		Output: state,
	}, nil
}

func (r AnsibleTaskList) Update(
	ctx context.Context,
	req infer.UpdateRequest[AnsibleTaskListArgs, AnsibleTaskListState],
) (infer.UpdateResponse[AnsibleTaskListState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/AnsibleTaskList.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:AnsibleTaskList"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	state := r.updateState(req.Inputs, req.State, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if req.DryRun {
		span.SetStatus(codes.Ok, "")
		return infer.UpdateResponse[AnsibleTaskListState]{
			Output: state,
		}, nil
	}

	var err error
	state, err = r.run(ctx, req.Inputs, state, "update")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[AnsibleTaskListState]{
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[AnsibleTaskListState]{
		Output: state,
	}, nil
}

func (r AnsibleTaskList) Delete(
	ctx context.Context,
	req infer.DeleteRequest[AnsibleTaskListState],
) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/AnsibleTaskList.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:AnsibleTaskList"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	_, err := r.run(ctx, req.State.AnsibleTaskListArgs, req.State, "delete")
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
