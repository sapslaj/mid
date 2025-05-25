package resource

import (
	"context"
	"errors"
	"fmt"

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

type Group struct{}

type GroupArgs struct {
	Name      *string              `pulumi:"name,optional"`
	Ensure    *string              `pulumi:"ensure,optional"`
	Force     *bool                `pulumi:"force,optional"`
	Gid       *int                 `pulumi:"gid,optional"`
	GidMax    *int                 `pulumi:"gidMax,optional"`
	GidMin    *int                 `pulumi:"gidMin,optional"`
	Local     *bool                `pulumi:"local,optional"`
	NonUnique *bool                `pulumi:"nonUnique,optional"`
	System    *bool                `pulumi:"system,optional"`
	Triggers  *types.TriggersInput `pulumi:"triggers,optional"`
}

type GroupState struct {
	GroupArgs
	Name     string               `pulumi:"name"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

func (r Group) argsToTaskParameters(input GroupArgs) (ansible.GroupParameters, error) {
	if input.Name == nil {
		return ansible.GroupParameters{}, errors.New("someone forgot to set the auto-named input.Name")
	}
	return ansible.GroupParameters{
		Force:     input.Force,
		Gid:       input.Gid,
		GidMax:    input.GidMax,
		GidMin:    input.GidMin,
		Local:     input.Local,
		Name:      *input.Name,
		NonUnique: input.NonUnique,
		State:     ansible.OptionalGroupState(input.Ensure),
		System:    input.System,
	}, nil
}

func (r Group) updateState(olds GroupState, news GroupArgs, changed bool) GroupState {
	olds.GroupArgs = news
	if news.Name != nil {
		olds.Name = *news.Name
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Group) Diff(
	ctx context.Context,
	id string,
	olds GroupState,
	news GroupArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Group.Diff", trace.WithAttributes(
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

	if news.Name == nil {
		news.Name = &olds.Name
	} else if *news.Name != olds.Name {
		diff.HasChanges = true
		diff.DetailedDiff["name"] = p.PropertyDiff{
			Kind:      p.UpdateReplace,
			InputDiff: true,
		}
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
			"ensure",
			"force",
			"gid",
			"gidMax",
			"gidMin",
			"local",
			"nonUnique",
			"system",
		}),
		types.DiffTriggers(olds, news),
	)

	span.SetStatus(codes.Ok, "")
	return diff, nil
}

func (r Group) Create(
	ctx context.Context,
	name string,
	input GroupArgs,
	preview bool,
) (string, GroupState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Group.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(GroupState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", state, err
	}
	span.SetAttributes(attribute.String("id", id))

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}
	call.Args.Check = preview

	if preview {
		canConnect, _ := executor.CanConnect(ctx, config.Connection, 4)
		if !canConnect {
			return id, state, nil
		}
	}

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, config.Connection, call)
	if err != nil || !callResult.Result.Success {
		err = fmt.Errorf(
			"creating group failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, state, nil
}

func (r Group) Read(
	ctx context.Context,
	id string,
	inputs GroupArgs,
	state GroupState,
) (string, GroupArgs, GroupState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Group.Read", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil {
		inputs.Name = ptr.Of(state.Name)
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}
	call.Args.Check = true

	canConnect, err := executor.CanConnect(ctx, config.Connection, 4)

	if !canConnect {
		return id, inputs, GroupState{
			GroupArgs: inputs,
		}, nil
	}

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, config.Connection, call)
	if err != nil || !callResult.Result.Success {
		err = fmt.Errorf(
			"reading group failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	result, err := ansible.GroupReturnFromRPCResult(callResult)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r Group) Update(
	ctx context.Context,
	id string,
	olds GroupState,
	news GroupArgs,
	preview bool,
) (GroupState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Group.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil {
		news.Name = ptr.Of(olds.Name)
	}

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}
	call.Args.Check = preview

	if preview {
		call.Args.Check = true
		canConnect, _ := executor.CanConnect(ctx, config.Connection, 4)
		if !canConnect {
			return olds, nil
		}
	}

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, config.Connection, call)
	if err != nil || !callResult.Result.Success {
		err = fmt.Errorf(
			"updating group failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	result, err := ansible.GroupReturnFromRPCResult(callResult)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	state := r.updateState(olds, news, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Group) Delete(
	ctx context.Context,
	id string,
	props GroupState,
) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:Group.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	if props.Ensure != nil && *props.Ensure == "absent" {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	args := props.GroupArgs
	args.Name = &props.Name

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	parameters.State = ansible.OptionalGroupState("absent")

	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection, 10)

	if !canConnect {
		if config.GetDeleteUnreachable() {
			return nil
		}

		if err == nil {
			err = fmt.Errorf("cannot connect to host")
		} else {
			err = fmt.Errorf("cannot connect to host: %w", err)
		}
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, config.Connection, call)
	if err != nil || !callResult.Result.Success {
		err = fmt.Errorf(
			"deleting group failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
