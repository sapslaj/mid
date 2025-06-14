package resource

import (
	"context"
	"errors"
	"slices"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
)

type Package struct{}

type PackageArgs struct {
	Name     *string              `pulumi:"name,optional"`
	Names    *[]string            `pulumi:"names,optional"`
	Ensure   *string              `pulumi:"ensure,optional"`
	Triggers *types.TriggersInput `pulumi:"triggers,optional"`
}

type PackageState struct {
	PackageArgs
	Ensure   string               `pulumi:"ensure"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

func (r Package) argsToTaskParameters(input PackageArgs) (ansible.PackageParameters, error) {
	parameters := ansible.PackageParameters{}
	if input.Ensure != nil {
		parameters.State = *input.Ensure
	} else {
		parameters.State = "present"
	}
	if input.Name == nil && input.Names == nil {
		return parameters, errors.New("either name or names but be provided")
	}
	if input.Names == nil {
		parameters.Name = *input.Name
	} else if len(*input.Names) == 1 {
		parameters.Name = (*input.Names)[0]
	} else {
		parameters.Name = *input.Names
	}
	return parameters, nil
}

func (r Package) updateState(inputs PackageArgs, state PackageState, changed bool) PackageState {
	state.PackageArgs = inputs
	if inputs.Ensure != nil {
		state.Ensure = *inputs.Ensure
	} else {
		state.Ensure = "present"
	}
	state.Triggers = types.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r Package) Diff(
	ctx context.Context,
	req infer.DiffRequest[PackageArgs, PackageState],
) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Package.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:Package"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: true,
	}

	if req.Inputs.Name != nil {
		if req.State.Name == nil {
			diff.HasChanges = true
			diff.DetailedDiff["name"] = p.PropertyDiff{
				Kind:      p.Add,
				InputDiff: true,
			}
		} else if *req.Inputs.Name != *req.State.Name {
			diff.HasChanges = true
			diff.DetailedDiff["name"] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
	}

	if req.Inputs.Names != nil {
		if req.State.Names == nil {
			diff.HasChanges = true
			diff.DetailedDiff["names"] = p.PropertyDiff{
				Kind:      p.Add,
				InputDiff: true,
			}
		} else if !slices.Equal(*req.State.Names, *req.Inputs.Names) {
			diff.HasChanges = true
			diff.DetailedDiff["names"] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
	} else if req.State.Names != nil && !slices.Equal(*req.State.Names, *req.Inputs.Names) {
		diff.HasChanges = true
		diff.DetailedDiff["names"] = p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: true,
		}
	}

	if req.Inputs.Ensure != nil && *req.Inputs.Ensure != req.State.Ensure {
		diff.HasChanges = true
		diff.DetailedDiff["ensure"] = p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: true,
		}
	}

	diff = types.MergeDiffResponses(diff, types.DiffTriggers(req.State, req.Inputs))

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r Package) Create(
	ctx context.Context,
	req infer.CreateRequest[PackageArgs],
) (infer.CreateResponse[PackageState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Package.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:Package"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(req.Inputs, PackageState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[PackageState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[PackageState]{
			ID:     id,
			Output: state,
		}, err
	}

	_, err = executor.AnsibleExecute[
		ansible.PackageParameters,
		ansible.PackageReturn,
	](ctx, config.Connection, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[PackageState]{
				ID:     id,
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[PackageState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[PackageState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r Package) Read(
	ctx context.Context,
	req infer.ReadRequest[PackageArgs, PackageState],
) (infer.ReadResponse[PackageArgs, PackageState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Package.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:Pacakge"),
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
		return infer.ReadResponse[PackageArgs, PackageState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.PackageParameters,
		ansible.PackageReturn,
	](ctx, config.Connection, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[PackageArgs, PackageState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[PackageArgs, PackageState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	if result.IsChanged() {
		if *req.Inputs.Ensure == "absent" {
			// we're going from present? to absent
			if state.Ensure == "absent" {
				state.Ensure = "present"
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[PackageArgs, PackageState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r Package) Update(
	ctx context.Context,
	req infer.UpdateRequest[PackageArgs, PackageState],
) (infer.UpdateResponse[PackageState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Package.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:Package"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if req.Inputs.Ensure != nil && *req.Inputs.Ensure == "absent" {
		parameters, err := r.argsToTaskParameters(req.Inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[PackageState]{
				Output: state,
			}, err
		}

		result, err := executor.AnsibleExecute[
			ansible.PackageParameters,
			ansible.PackageReturn,
		](ctx, config.Connection, parameters, req.DryRun)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.UpdateResponse[PackageState]{
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[PackageState]{
				Output: state,
			}, err
		}

		state = r.updateState(req.Inputs, state, result.IsChanged())

		span.SetStatus(codes.Ok, "")
		return infer.UpdateResponse[PackageState]{
			Output: state,
		}, nil
	}

	packageStateMap := map[string]string{}

	newState := state.Ensure
	if req.Inputs.Ensure != nil {
		newState = *req.Inputs.Ensure
	}

	if req.Inputs.Name != nil {
		packageStateMap[*req.Inputs.Name] = newState
	} else if req.Inputs.Names != nil {
		for _, name := range *req.Inputs.Names {
			packageStateMap[name] = newState
		}
	} else if state.Name != nil {
		packageStateMap[*state.Name] = newState
	} else if state.Names != nil {
		for _, name := range *state.Names {
			packageStateMap[name] = newState
		}
	} else {
		err := errors.New("we somehow forgot the package name, oops")
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[PackageState]{
			Output: state,
		}, err
	}

	if state.Name != nil {
		if _, exists := packageStateMap[*state.Name]; !exists {
			packageStateMap[*state.Name] = "absent"
		}
	} else if state.Names != nil {
		for _, name := range *state.Names {
			if _, exists := packageStateMap[name]; !exists {
				packageStateMap[name] = "absent"
			}
		}
	}

	absents := []string{}
	presents := []string{}

	for name, packageState := range packageStateMap {
		if packageState == "absent" {
			absents = append(absents, name)
		} else {
			presents = append(presents, name)
		}
	}

	changed := false

	if len(absents) > 0 {
		parameters := ansible.PackageParameters{
			Name:  absents,
			State: "absent",
		}
		result, err := executor.AnsibleExecute[
			ansible.PackageParameters,
			ansible.PackageReturn,
		](ctx, config.Connection, parameters, req.DryRun)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.UpdateResponse[PackageState]{
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[PackageState]{
				Output: state,
			}, err
		}
		if result.IsChanged() {
			changed = true
		}
	}

	if len(presents) > 0 {
		parameters := ansible.PackageParameters{
			Name:  presents,
			State: newState,
		}
		result, err := executor.AnsibleExecute[
			ansible.PackageParameters,
			ansible.PackageReturn,
		](ctx, config.Connection, parameters, req.DryRun)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.UpdateResponse[PackageState]{
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[PackageState]{
				Output: state,
			}, err
		}
		if result.IsChanged() {
			changed = true
		}
	}

	state = r.updateState(req.Inputs, state, changed)
	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[PackageState]{
		Output: state,
	}, nil
}

func (r Package) Delete(ctx context.Context, req infer.DeleteRequest[PackageState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Package.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:Package"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	if req.State.Ensure == "absent" {
		return infer.DeleteResponse{}, nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(req.State.PackageArgs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}
	parameters.State = "absent"

	_, err = executor.AnsibleExecute[
		ansible.PackageParameters,
		ansible.PackageReturn,
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
