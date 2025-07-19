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
	"github.com/sapslaj/mid/provider/midtypes"
)

type Service struct{}

type ServiceArgs struct {
	Arguments  *string                  `pulumi:"arguments,optional"`
	Enabled    *bool                    `pulumi:"enabled,optional"`
	Name       string                   `pulumi:"name"`
	Pattern    *string                  `pulumi:"pattern,optional"`
	Runlevel   *string                  `pulumi:"runlevel,optional"`
	Sleep      *int                     `pulumi:"sleep,optional"`
	State      *string                  `pulumi:"state,optional"`
	Use        *string                  `pulumi:"use,optional"`
	Connection *midtypes.Connection     `pulumi:"connection,optional"`
	Config     *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers   *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type ServiceState struct {
	ServiceArgs
	Triggers midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r Service) argsToTaskParameters(input ServiceArgs) (ansible.ServiceParameters, error) {
	return ansible.ServiceParameters{
		Arguments: input.Arguments,
		Enabled:   input.Enabled,
		Name:      input.Name,
		Pattern:   input.Pattern,
		Runlevel:  input.Runlevel,
		Sleep:     input.Sleep,
		State:     ansible.OptionalServiceState(input.State),
		Use:       input.Use,
	}, nil
}

func (r Service) updateState(inputs ServiceArgs, state ServiceState, changed bool) ServiceState {
	state.ServiceArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r Service) Diff(
	ctx context.Context,
	req infer.DiffRequest[ServiceArgs, ServiceState],
) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Service.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:Service"),
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
			"arguments",
			"enabled",
			"name",
			"pattern",
			"runlevel",
			"sleep",
			"state",
			"use",
		}),
		midtypes.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r Service) Create(
	ctx context.Context,
	req infer.CreateRequest[ServiceArgs],
) (infer.CreateResponse[ServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Service.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:Service"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(req.Inputs, ServiceState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[ServiceState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[ServiceState]{
			ID:     id,
			Output: state,
		}, err
	}

	_, err = executor.AnsibleExecute[
		ansible.ServiceParameters,
		ansible.ServiceReturn,
	](ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[ServiceState]{
				ID:     id,
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[ServiceState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[ServiceState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r Service) Read(
	ctx context.Context,
	req infer.ReadRequest[ServiceArgs, ServiceState],
) (infer.ReadResponse[ServiceArgs, ServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Service.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:Service"),
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
		return infer.ReadResponse[ServiceArgs, ServiceState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.ServiceParameters,
		ansible.ServiceReturn,
	](ctx, connection, config, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[ServiceArgs, ServiceState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[ServiceArgs, ServiceState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[ServiceArgs, ServiceState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r Service) Update(
	ctx context.Context,
	req infer.UpdateRequest[ServiceArgs, ServiceState],
) (infer.UpdateResponse[ServiceState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Service.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:Service"),
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
		return infer.UpdateResponse[ServiceState]{
			Output: state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.ServiceParameters,
		ansible.ServiceReturn,
	](ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[ServiceState]{
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[ServiceState]{
			Output: state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[ServiceState]{
		Output: state,
	}, nil
}

func (r Service) Delete(ctx context.Context, req infer.DeleteRequest[ServiceState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Service.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:Service"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.State.Connection)
	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	args := ServiceArgs{
		Arguments: req.State.Arguments,
		Enabled:   req.State.Enabled,
		Name:      req.State.Name,
		Pattern:   req.State.Pattern,
		Runlevel:  req.State.Runlevel,
		Sleep:     req.State.Sleep,
		State:     req.State.State,
		Use:       req.State.Use,
	}

	runPlay := false

	if args.Enabled != nil && *args.Enabled {
		runPlay = true
		args.Enabled = ptr.Of(false)
	}
	if args.State != nil && *args.State != "stopped" {
		runPlay = true
		args.State = ptr.Of("stopped")
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

	_, err = executor.AnsibleExecute[
		ansible.ServiceParameters,
		ansible.ServiceReturn,
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
