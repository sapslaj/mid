package resource

import (
	"context"
	"errors"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
)

type SystemdService struct{}

type SystemdServiceArgs struct {
	DaemonReexec *bool                `pulumi:"daemonReexec,optional"`
	DaemonReload *bool                `pulumi:"daemonReload,optional"`
	Enabled      *bool                `pulumi:"enabled,optional"`
	Force        *bool                `pulumi:"force,optional"`
	Masked       *bool                `pulumi:"masked,optional"`
	Name         *string              `pulumi:"name,optional"`
	NoBlock      *bool                `pulumi:"noBlock,optional"`
	Scope        *string              `pulumi:"scope,optional"`
	Ensure       *string              `pulumi:"ensure,optional"` // TODO: enum for this?
	Triggers     *types.TriggersInput `pulumi:"triggers,optional"`
}

type SystemdServiceState struct {
	SystemdServiceArgs
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

func (r SystemdService) argsToTaskParameters(input SystemdServiceArgs) (ansible.SystemdServiceParameters, error) {
	return ansible.SystemdServiceParameters{
		DaemonReexec: input.DaemonReexec,
		DaemonReload: input.DaemonReload,
		Enabled:      input.Enabled,
		Force:        input.Force,
		Masked:       input.Masked,
		Name:         input.Name,
		NoBlock:      input.NoBlock,
		Scope:        ansible.OptionalSystemdServiceScope(input.Scope),
		State:        ansible.OptionalSystemdServiceState(input.Ensure),
	}, nil
}

func (r SystemdService) updateState(
	inputs SystemdServiceArgs,
	state SystemdServiceState,
	changed bool,
) SystemdServiceState {
	state.SystemdServiceArgs = inputs
	state.Triggers = types.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
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

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(req.State, req.Inputs, []string{
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
		types.DiffTriggers(req.State, req.Inputs),
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

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(req.Inputs, SystemdServiceState{}, true)
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

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[SystemdServiceState]{
			ID:     id,
			Output: state,
		}, err
	}

	if req.DryRun {
		systemdInfo, err := executor.AnsibleExecute[
			ansible.SystemdInfoParameters,
			ansible.SystemdInfoReturn,
		](ctx, config.Connection, ansible.SystemdInfoParameters{}, true)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.CreateResponse[SystemdServiceState]{
					ID:     id,
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.CreateResponse[SystemdServiceState]{
				ID:     id,
				Output: state,
			}, err
		}
		if systemdInfo.Units == nil {
			// some reason couldn't get unit list, assume that it will be fine.
			// TODO: log warning?
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[SystemdServiceState]{
				ID:     id,
				Output: state,
			}, nil
		}
		_, unitPresent := (*systemdInfo.Units)[*req.Inputs.Name]
		if !unitPresent {
			// Unit isn't present during req.DryRun, which might be expected.
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[SystemdServiceState]{
				ID:     id,
				Output: state,
			}, nil
		}
	}

	_, err = executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, config.Connection, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[SystemdServiceState]{
				ID:     id,
				Output: state,
			}, nil
		}
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

	config := infer.GetConfig[types.Config](ctx)

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
	](ctx, config.Connection, parameters, true)
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

func (r SystemdService) Update(ctx context.Context, req infer.UpdateRequest[SystemdServiceArgs, SystemdServiceState]) (infer.UpdateResponse[SystemdServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[SystemdServiceState]{
			Output: state,
		}, err
	}

	refresh := false
	triggerDiff := types.DiffTriggers(req.State, req.Inputs)
	if triggerDiff.HasChanges {
		refresh = true
	}

	if refresh && req.Inputs.Ensure != nil && *req.Inputs.Ensure == "started" {
		parameters.State = ansible.OptionalSystemdServiceState("restarted")
	}

	if req.DryRun {
		systemdInfo, err := executor.AnsibleExecute[
			ansible.SystemdInfoParameters,
			ansible.SystemdInfoReturn,
		](ctx, config.Connection, ansible.SystemdInfoParameters{}, true)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.UpdateResponse[SystemdServiceState]{
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[SystemdServiceState]{
				Output: state,
			}, err
		}
		if systemdInfo.Units == nil {
			// some reason couldn't get unit list, assume that it will be fine.
			// TODO: log warning?
			state = r.updateState(req.Inputs, state, true)
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[SystemdServiceState]{
				Output: state,
			}, nil
		}
		_, unitPresent := (*systemdInfo.Units)[*req.Inputs.Name]
		if !unitPresent {
			// Unit isn't present during dry run, which might be expected.
			span.SetStatus(codes.Ok, "")
			state = r.updateState(req.Inputs, state, true)
			return infer.UpdateResponse[SystemdServiceState]{
				Output: state,
			}, nil
		}
	}

	result, err := executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, config.Connection, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[SystemdServiceState]{
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[SystemdServiceState]{
			Output: state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[SystemdServiceState]{
		Output: state,
	}, nil
}

func (r SystemdService) Delete(ctx context.Context, req infer.DeleteRequest[SystemdServiceState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/SystemdService.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:SystemdService"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

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

	runPlay := false

	if args.Enabled != nil && *args.Enabled {
		runPlay = true
		args.Enabled = ptr.Of(false)
	}
	if args.Ensure != nil && *args.Ensure != "stopped" {
		runPlay = true
		args.Ensure = ptr.Of("stopped")
	}

	span.SetAttributes(attribute.Bool("run_play", runPlay))

	if !runPlay {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}

	systemdInfo, err := executor.AnsibleExecute[
		ansible.SystemdInfoParameters,
		ansible.SystemdInfoReturn,
	](ctx, config.Connection, ansible.SystemdInfoParameters{}, true)
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
	if systemdInfo.Units == nil {
		// some reason couldn't get unit list, assume that it will be fine.
		// TODO: log warning?
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}
	_, unitPresent := (*systemdInfo.Units)[*req.State.Name]
	if !unitPresent {
		// Unit might have been removed from system. In this case it is okay to
		// delete from req.State.
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	_, err = executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, config.Connection, parameters, false)
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
