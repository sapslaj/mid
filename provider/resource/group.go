package resource

import (
	"context"
	"errors"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type Group struct{}

type GroupArgs struct {
	Name       string                   `pulumi:"name"`
	Ensure     *string                  `pulumi:"ensure,optional"`
	Force      *bool                    `pulumi:"force,optional"`
	Gid        *int                     `pulumi:"gid,optional"`
	GidMax     *int                     `pulumi:"gidMax,optional"`
	GidMin     *int                     `pulumi:"gidMin,optional"`
	Local      *bool                    `pulumi:"local,optional"`
	NonUnique  *bool                    `pulumi:"nonUnique,optional"`
	System     *bool                    `pulumi:"system,optional"`
	Connection *midtypes.Connection     `pulumi:"connection,optional"`
	Config     *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers   *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type GroupState struct {
	GroupArgs
	Triggers midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r Group) argsToTaskParameters(input GroupArgs) (ansible.GroupParameters, error) {
	return ansible.GroupParameters{
		Force:     input.Force,
		Gid:       input.Gid,
		GidMax:    input.GidMax,
		GidMin:    input.GidMin,
		Local:     input.Local,
		Name:      input.Name,
		NonUnique: input.NonUnique,
		State:     ansible.OptionalGroupState(input.Ensure),
		System:    input.System,
	}, nil
}

func (r Group) updateState(inputs GroupArgs, state GroupState, changed bool) GroupState {
	state.GroupArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r Group) Diff(ctx context.Context, req infer.DiffRequest[GroupArgs, GroupState]) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Group.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:Group"),
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

	if req.Inputs.Name != req.State.Name {
		diff.HasChanges = true
		diff.DetailedDiff["path"] = p.PropertyDiff{
			Kind:      p.UpdateReplace,
			InputDiff: true,
		}
	}

	diff = midtypes.MergeDiffResponses(
		diff,
		midtypes.DiffAttributes(req.State, req.Inputs, []string{
			"ensure",
			"force",
			"gid",
			"gidMax",
			"gidMin",
			"local",
			"nonUnique",
			"system",
		}),
		midtypes.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r Group) Create(
	ctx context.Context,
	req infer.CreateRequest[GroupArgs],
) (infer.CreateResponse[GroupState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Group.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:Group"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(req.Inputs, GroupState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[GroupState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[GroupState]{
			ID:     id,
			Output: state,
		}, err
	}

	_, err = executor.AnsibleExecute[
		ansible.GroupParameters,
		ansible.GroupReturn,
	](ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[GroupState]{
				ID:     id,
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[GroupState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[GroupState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r Group) Read(
	ctx context.Context,
	req infer.ReadRequest[GroupArgs, GroupState],
) (infer.ReadResponse[GroupArgs, GroupState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Group.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:Group"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[GroupArgs, GroupState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.GroupParameters,
		ansible.GroupReturn,
	](ctx, connection, config, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[GroupArgs, GroupState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[GroupArgs, GroupState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[GroupArgs, GroupState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r Group) Update(
	ctx context.Context,
	req infer.UpdateRequest[GroupArgs, GroupState],
) (infer.UpdateResponse[GroupState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Group.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:Group"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[GroupState]{
			Output: state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.GroupParameters,
		ansible.GroupReturn,
	](ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[GroupState]{
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[GroupState]{
			Output: state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[GroupState]{
		Output: state,
	}, nil
}

func (r Group) Delete(ctx context.Context, req infer.DeleteRequest[GroupState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Group.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:Group"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	if req.State.Ensure != nil && *req.State.Ensure == "absent" {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	connection := midtypes.GetConnection(ctx, req.State.Connection)
	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	parameters, err := r.argsToTaskParameters(req.State.GroupArgs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}
	parameters.State = ansible.OptionalGroupState("absent")

	_, err = executor.AnsibleExecute[
		ansible.GroupParameters,
		ansible.GroupReturn,
	](ctx, connection, config, parameters, false)
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
