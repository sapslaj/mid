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
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type User struct{}

type UserArgs struct {
	// TODO: support more features
	Name            string                   `pulumi:"name"`
	Ensure          *string                  `pulumi:"ensure,optional"`
	GroupsExclusive *bool                    `pulumi:"groupsExclusive,optional"`
	Comment         *string                  `pulumi:"comment,optional"`
	Local           *bool                    `pulumi:"local,optional"`
	Group           *string                  `pulumi:"group,optional"`
	Groups          *[]string                `pulumi:"groups,optional"`
	Home            *string                  `pulumi:"home,optional"`
	Force           *bool                    `pulumi:"force,optional"`
	ManageHome      *bool                    `pulumi:"manageHome,optional"`
	NonUnique       *bool                    `pulumi:"nonUnique,optional"`
	Password        *string                  `pulumi:"password,optional"`
	Shell           *string                  `pulumi:"shell,optional"`
	Skeleton        *string                  `pulumi:"skeleton,optional"`
	System          *bool                    `pulumi:"system,optional"`
	Uid             *int                     `pulumi:"uid,optional"`
	UidMax          *int                     `pulumi:"uidMax,optional"`
	UidMin          *int                     `pulumi:"uidMin,optional"`
	Umask           *string                  `pulumi:"umask,optional"`
	UpdatePassword  *string                  `pulumi:"updatePassword,optional"`
	Connection      *midtypes.Connection     `pulumi:"connection,optional"`
	Config          *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers        *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type UserState struct {
	UserArgs
	Triggers midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r User) argsToTaskParameters(input UserArgs) (ansible.UserParameters, error) {
	groupsExclusive := false
	if input.GroupsExclusive != nil {
		groupsExclusive = *input.GroupsExclusive
	}
	return ansible.UserParameters{
		Name:           input.Name,
		State:          ansible.OptionalUserState(input.Ensure),
		Append:         ptr.Of(!groupsExclusive),
		Comment:        input.Comment,
		Local:          input.Local,
		Group:          input.Group,
		Groups:         input.Groups,
		Home:           input.Home,
		Force:          input.Force,
		CreateHome:     input.ManageHome,
		MoveHome:       input.ManageHome,
		Remove:         input.ManageHome,
		NonUnique:      input.NonUnique,
		Password:       input.Password,
		Shell:          input.Shell,
		Skeleton:       input.Skeleton,
		System:         input.System,
		Uid:            input.Uid,
		UidMax:         input.UidMax,
		UidMin:         input.UidMin,
		Umask:          input.Umask,
		UpdatePassword: ansible.OptionalUserUpdatePassword(input.UpdatePassword),
	}, nil
}

func (r User) updateState(inputs UserArgs, state UserState, changed bool) UserState {
	state.UserArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r User) Diff(ctx context.Context, req infer.DiffRequest[UserArgs, UserState]) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/User.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:User"),
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
			"groupsExclusive",
			"comment",
			"local",
			"group",
			"groups",
			"home",
			"force",
			"manageHome",
			"nonUnique",
			"password",
			"shell",
			"skeleton",
			"system",
			"uid",
			"uidMax",
			"uidMin",
			"umask",
			"updatePassword",
		}),
		midtypes.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r User) Create(
	ctx context.Context,
	req infer.CreateRequest[UserArgs],
) (infer.CreateResponse[UserState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/User.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:User"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(req.Inputs, UserState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[UserState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[UserState]{
			ID:     id,
			Output: state,
		}, err
	}

	_, err = executor.AnsibleExecute[
		ansible.UserParameters,
		ansible.UserReturn,
	](ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[UserState]{
				ID:     id,
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[UserState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[UserState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r User) Read(
	ctx context.Context,
	req infer.ReadRequest[UserArgs, UserState],
) (infer.ReadResponse[UserArgs, UserState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/User.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:User"),
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
		return infer.ReadResponse[UserArgs, UserState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.UserParameters,
		ansible.UserReturn,
	](ctx, connection, config, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[UserArgs, UserState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[UserArgs, UserState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[UserArgs, UserState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r User) Update(
	ctx context.Context,
	req infer.UpdateRequest[UserArgs, UserState],
) (infer.UpdateResponse[UserState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/User.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:User"),
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
		return infer.UpdateResponse[UserState]{
			Output: state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.UserParameters,
		ansible.UserReturn,
	](ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[UserState]{
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[UserState]{
			Output: state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[UserState]{
		Output: state,
	}, nil
}

func (r User) Delete(ctx context.Context, req infer.DeleteRequest[UserState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/User.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:User"),
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

	parameters, err := r.argsToTaskParameters(req.State.UserArgs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}
	parameters.State = ansible.OptionalUserState("absent")

	_, err = executor.AnsibleExecute[
		ansible.UserParameters,
		ansible.UserReturn,
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
