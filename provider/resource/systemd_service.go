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
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type SystemdService struct{}

type SystemdServiceEnsure string

const (
	SystemdServiceEnsureStarted   SystemdServiceEnsure = "started"
	SystemdServiceEnsureStopped   SystemdServiceEnsure = "stopped"
	SystemdServiceEnsureReloaded  SystemdServiceEnsure = "reloaded"
	SystemdServiceEnsureRestarted SystemdServiceEnsure = "restarted"
)

type SystemdServiceArgs struct {
	DaemonReexec *bool                    `pulumi:"daemonReexec,optional"`
	DaemonReload *bool                    `pulumi:"daemonReload,optional"`
	Enabled      *bool                    `pulumi:"enabled,optional"`
	Force        *bool                    `pulumi:"force,optional"`
	Masked       *bool                    `pulumi:"masked,optional"`
	Name         *string                  `pulumi:"name,optional"`
	NoBlock      *bool                    `pulumi:"noBlock,optional"`
	Scope        *string                  `pulumi:"scope,optional"`
	Ensure       *SystemdServiceEnsure    `pulumi:"ensure,optional"`
	Connection   *midtypes.Connection     `pulumi:"connection,optional"`
	Config       *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers     *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type SystemdServiceState struct {
	SystemdServiceArgs
	Triggers midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r SystemdService) argsToTaskParameters(input SystemdServiceArgs) (ansible.SystemdServiceParameters, error) {
	var state *ansible.SystemdServiceState
	if input.Ensure != nil {
		state = ansible.OptionalSystemdServiceState(string(*input.Ensure))
	}
	return ansible.SystemdServiceParameters{
		DaemonReexec: input.DaemonReexec,
		DaemonReload: input.DaemonReload,
		Enabled:      input.Enabled,
		Force:        input.Force,
		Masked:       input.Masked,
		Name:         input.Name,
		NoBlock:      input.NoBlock,
		Scope:        ansible.OptionalSystemdServiceScope(input.Scope),
		State:        state,
	}, nil
}

func (r SystemdService) updateState(
	inputs SystemdServiceArgs,
	state SystemdServiceState,
	changed bool,
) SystemdServiceState {
	state.SystemdServiceArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r SystemdService) doesUnitExist(
	ctx context.Context,
	connection midtypes.Connection,
	config midtypes.ResourceConfig,
	name string,
) (bool, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.doesUnitExist", trace.WithAttributes(
		attribute.String("connection.host", *connection.Host),
		attribute.String("name", name),
	))
	defer span.End()

	result, err := executor.CallAgent[
		rpc.SystemdUnitShortStatusArgs,
		rpc.SystemdUnitShortStatusResult,
	](ctx, connection, config, rpc.RPCCall[rpc.SystemdUnitShortStatusArgs]{
		RPCFunction: rpc.RPCSystemdUnitShortStatus,
		Args: rpc.SystemdUnitShortStatusArgs{
			Name: name,
		},
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	span.SetStatus(codes.Ok, "")
	return result.Result.Exists, nil
}

func (r SystemdService) updateService(
	ctx context.Context,
	inputs SystemdServiceArgs,
	state SystemdServiceState,
	dryRun bool,
) (SystemdServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.updateService", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()

	defer span.SetAttributes(telemetry.OtelJSON("state", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	refresh := false
	if state.Triggers.Refresh != nil || inputs.Triggers != nil {
		triggerDiff := midtypes.DiffTriggers(state, inputs)
		if triggerDiff.HasChanges {
			refresh = true
		}
	}

	span.SetAttributes(attribute.Bool("refresh", refresh))

	if refresh && inputs.Ensure != nil && *inputs.Ensure == "started" {
		parameters.State = ansible.OptionalSystemdServiceState(string(SystemdServiceEnsureRestarted))
	}

	if dryRun && inputs.Name != nil {
		unitPresent, err := r.doesUnitExist(ctx, connection, config, *inputs.Name)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && dryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return state, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if !unitPresent {
			// Unit isn't present during dry run, which might be expected.
			span.SetStatus(codes.Ok, "")
			state = r.updateState(inputs, state, true)
			return state, nil
		}
	}

	result, err := executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, connection, config, parameters, dryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && dryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return state, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state = r.updateState(inputs, state, result.IsChanged())

	return state, nil
}

func (r SystemdService) Diff(
	ctx context.Context,
	req infer.DiffRequest[SystemdServiceArgs, SystemdServiceState],
) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
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

	diff = midtypes.MergeDiffResponses(
		diff,
		midtypes.DiffAttributes(req.State, req.Inputs, []string{
			"daemonReexec",
			"daemonReload",
			"enabled",
			"ensure",
			"force",
			"masked",
			"name",
			"noBlock",
			"scope",
		}),
		midtypes.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r SystemdService) Create(
	ctx context.Context,
	req infer.CreateRequest[SystemdServiceArgs],
) (infer.CreateResponse[SystemdServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	state := SystemdServiceState{
		SystemdServiceArgs: req.Inputs,
	}
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[SystemdServiceState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("id", id))

	state, err = r.updateService(ctx, req.Inputs, state, req.DryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[SystemdServiceState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[SystemdServiceState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r SystemdService) Read(
	ctx context.Context,
	req infer.ReadRequest[SystemdServiceArgs, SystemdServiceState],
) (infer.ReadResponse[SystemdServiceArgs, SystemdServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
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
		return infer.ReadResponse[SystemdServiceArgs, SystemdServiceState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, connection, config, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[SystemdServiceArgs, SystemdServiceState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[SystemdServiceArgs, SystemdServiceState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[SystemdServiceArgs, SystemdServiceState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r SystemdService) Update(
	ctx context.Context,
	req infer.UpdateRequest[SystemdServiceArgs, SystemdServiceState],
) (infer.UpdateResponse[SystemdServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	var err error
	state, err = r.updateService(ctx, req.Inputs, state, req.DryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[SystemdServiceState]{
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[SystemdServiceState]{
		Output: state,
	}, nil
}

func (r SystemdService) Delete(
	ctx context.Context,
	req infer.DeleteRequest[SystemdServiceState],
) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.State.Connection)
	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	args := SystemdServiceArgs{
		DaemonReexec: req.State.DaemonReexec,
		DaemonReload: req.State.DaemonReload,
		Enabled:      req.State.Enabled,
		Force:        req.State.Force,
		Masked:       req.State.Masked,
		Name:         req.State.Name,
		NoBlock:      req.State.NoBlock,
		Scope:        req.State.Scope,
		Ensure:       req.State.Ensure,
	}

	takeAction := false

	if args.Enabled != nil && *args.Enabled {
		takeAction = true
		args.Enabled = ptr.Of(false)
	}

	if args.Ensure != nil && *args.Ensure != SystemdServiceEnsureStopped {
		takeAction = true
		args.Ensure = ptr.Of(SystemdServiceEnsureStopped)
	}

	if args.Name == nil {
		takeAction = false
	}

	span.SetAttributes(attribute.Bool("take_action", takeAction))

	if !takeAction {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}

	unitPresent, err := r.doesUnitExist(ctx, connection, config, *args.Name)
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

	span.SetAttributes(attribute.Bool("unit_present", unitPresent))
	if !unitPresent {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	_, err = executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
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
