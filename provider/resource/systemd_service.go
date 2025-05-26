package resource

import (
	"context"
	"errors"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"
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

func (r SystemdService) updateState(olds SystemdServiceState, news SystemdServiceArgs, changed bool) SystemdServiceState {
	olds.SystemdServiceArgs = news
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r SystemdService) Diff(
	ctx context.Context,
	id string,
	olds SystemdServiceState,
	news SystemdServiceArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:SystemdService.Diff", trace.WithAttributes(
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

	if news.Name == nil && olds.Name != nil {
		news.Name = olds.Name
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
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
		types.DiffTriggers(olds, news),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r SystemdService) Create(
	ctx context.Context,
	name string,
	input SystemdServiceArgs,
	preview bool,
) (string, SystemdServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:SystemdService.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil && (input.Enabled != nil || input.Masked != nil || input.Ensure != nil) {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(SystemdServiceState{}, input, true)

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

	if preview {
		systemdInfo, err := executor.AnsibleExecute[
			ansible.SystemdInfoParameters,
			ansible.SystemdInfoReturn,
		](ctx, config.Connection, ansible.SystemdInfoParameters{}, true)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && preview {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return id, state, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return id, state, err
		}
		if systemdInfo.Units == nil {
			// some reason couldn't get unit list, assume that it will be fine.
			// TODO: log warning?
			span.SetStatus(codes.Ok, "")
			return id, state, nil
		}
		_, unitPresent := (*systemdInfo.Units)[*input.Name]
		if !unitPresent {
			// Unit isn't present during preview, which might be expected.
			span.SetStatus(codes.Ok, "")
			return id, state, nil
		}
	}

	_, err = executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, config.Connection, parameters, preview)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && preview {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return id, state, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, state, nil
}

func (r SystemdService) Read(
	ctx context.Context,
	id string,
	inputs SystemdServiceArgs,
	state SystemdServiceState,
) (string, SystemdServiceArgs, SystemdServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:SystemdService.Read", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil && state.Name != nil && (inputs.Enabled != nil || inputs.Masked != nil || inputs.Ensure != nil) {
		inputs.Name = state.Name
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	result, err := executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, config.Connection, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return id, inputs, SystemdServiceState{
				SystemdServiceArgs: inputs,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r SystemdService) Update(
	ctx context.Context,
	id string,
	olds SystemdServiceState,
	news SystemdServiceArgs,
	preview bool,
) (SystemdServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:SystemdService.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil && olds.Name != nil && (news.Enabled != nil || news.Masked != nil || news.Ensure != nil) {
		news.Name = olds.Name
	}

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	refresh := false
	triggerDiff := types.DiffTriggers(olds, news)
	if triggerDiff.HasChanges {
		refresh = true
	}

	if refresh && news.Ensure != nil && *news.Ensure == "started" {
		parameters.State = ansible.OptionalSystemdServiceState("restarted")
	}

	if preview {
		systemdInfo, err := executor.AnsibleExecute[
			ansible.SystemdInfoParameters,
			ansible.SystemdInfoReturn,
		](ctx, config.Connection, ansible.SystemdInfoParameters{}, true)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && preview {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return olds, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return olds, err
		}
		if systemdInfo.Units == nil {
			// some reason couldn't get unit list, assume that it will be fine.
			// TODO: log warning?
			state := r.updateState(olds, news, true)
			span.SetStatus(codes.Ok, "")
			return state, nil
		}
		_, unitPresent := (*systemdInfo.Units)[*news.Name]
		if !unitPresent {
			// Unit isn't present during preview, which might be expected.
			span.SetStatus(codes.Ok, "")
			state := r.updateState(olds, news, true)
			return state, nil
		}
	}

	result, err := executor.AnsibleExecute[
		ansible.SystemdServiceParameters,
		ansible.SystemdServiceReturn,
	](ctx, config.Connection, parameters, preview)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && preview {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return olds, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	state := r.updateState(olds, news, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r SystemdService) Delete(
	ctx context.Context,
	id string,
	props SystemdServiceState,
) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:SystemdService.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	args := SystemdServiceArgs{
		DaemonReexec: props.DaemonReexec,
		DaemonReload: props.DaemonReload,
		Enabled:      props.Enabled,
		Force:        props.Force,
		Masked:       props.Masked,
		Name:         props.Name,
		NoBlock:      props.NoBlock,
		Scope:        props.Scope,
		Ensure:       props.Ensure,
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
		return nil
	}

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
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
			return nil
		}
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if systemdInfo.Units == nil {
		// some reason couldn't get unit list, assume that it will be fine.
		// TODO: log warning?
		span.SetStatus(codes.Ok, "")
		return nil
	}
	_, unitPresent := (*systemdInfo.Units)[*props.Name]
	if !unitPresent {
		// Unit might have been removed from system. In this case it is okay to
		// delete from state.
		span.SetStatus(codes.Ok, "")
		return nil
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
			return nil
		}
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
