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

type Service struct{}

type ServiceArgs struct {
	Arguments *string              `pulumi:"arguments,optional"`
	Enabled   *bool                `pulumi:"enabled,optional"`
	Name      *string              `pulumi:"name,optional"`
	Pattern   *string              `pulumi:"pattern,optional"`
	Runlevel  *string              `pulumi:"runlevel,optional"`
	Sleep     *int                 `pulumi:"sleep,optional"`
	State     *string              `pulumi:"state,optional"`
	Use       *string              `pulumi:"use,optional"`
	Triggers  *types.TriggersInput `pulumi:"triggers,optional"`
}

type ServiceState struct {
	ServiceArgs
	Name     string               `pulumi:"name"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

func (r Service) argsToTaskParameters(input ServiceArgs) (ansible.ServiceParameters, error) {
	if input.Name == nil {
		return ansible.ServiceParameters{}, errors.New("someone forgot to set the auto-named input.Name")
	}
	return ansible.ServiceParameters{
		Arguments: input.Arguments,
		Enabled:   input.Enabled,
		Name:      *input.Name,
		Pattern:   input.Pattern,
		Runlevel:  input.Runlevel,
		Sleep:     input.Sleep,
		State:     ansible.OptionalServiceState(input.State),
		Use:       input.Use,
	}, nil
}

func (r Service) updateState(olds ServiceState, news ServiceArgs, changed bool) ServiceState {
	olds.ServiceArgs = news
	if news.Name != nil {
		olds.Name = *news.Name
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Service) Diff(
	ctx context.Context,
	id string,
	olds ServiceState,
	news ServiceArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Service.Diff", trace.WithAttributes(
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
			"arguments",
			"enabled",
			"pattern",
			"runlevel",
			"sleep",
			"state",
			"use",
		}),
		types.DiffTriggers(olds, news),
	)

	span.SetStatus(codes.Ok, "")
	return diff, nil
}

func (r Service) Create(
	ctx context.Context,
	name string,
	input ServiceArgs,
	preview bool,
) (string, ServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Service.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(ServiceState{}, input, true)

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
			"service update failed: stderr=%s stdout=%s, err=%w",
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

func (r Service) Read(
	ctx context.Context,
	id string,
	inputs ServiceArgs,
	state ServiceState,
) (string, ServiceArgs, ServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Service.Read", trace.WithAttributes(
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
		return id, inputs, ServiceState{
			ServiceArgs: inputs,
		}, nil
	}

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, config.Connection, call)
	if err != nil || !callResult.Result.Success {
		err = fmt.Errorf(
			"service update failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	result, err := ansible.ServiceReturnFromRPCResult(callResult)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r Service) Update(
	ctx context.Context,
	id string,
	olds ServiceState,
	news ServiceArgs,
	preview bool,
) (ServiceState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Service.Update", trace.WithAttributes(
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
			"service update failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	result, err := ansible.ServiceReturnFromRPCResult(callResult)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	state := r.updateState(olds, news, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Service) Delete(
	ctx context.Context,
	id string,
	props ServiceState,
) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:Service.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	args := ServiceArgs{
		Arguments: props.Arguments,
		Enabled:   props.Enabled,
		Name:      &props.Name,
		Pattern:   props.Pattern,
		Runlevel:  props.Runlevel,
		Sleep:     props.Sleep,
		State:     props.State,
		Use:       props.Use,
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
		return nil
	}

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

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
			"service update failed: stderr=%s stdout=%s, err=%w",
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
