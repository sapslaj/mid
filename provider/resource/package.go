package resource

import (
	"context"
	"errors"
	"slices"

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

type Package struct{}

type PackageArgs struct {
	Name     *string              `pulumi:"name,optional"`
	Names    *[]string            `pulumi:"names,optional"`
	Ensure   *string              `pulumi:"ensure,optional"`
	Triggers *types.TriggersInput `pulumi:"triggers,optional"`
}

type PackageState struct {
	Name     *string              `pulumi:"name,optional"`
	Names    *[]string            `pulumi:"names,optional"`
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

func (r Package) updateState(olds PackageState, news PackageArgs, changed bool) PackageState {
	if news.Name != nil || news.Names != nil {
		olds.Name = news.Name
		olds.Names = news.Names
	}
	if news.Ensure != nil {
		olds.Ensure = *news.Ensure
	} else {
		olds.Ensure = "present"
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Package) Diff(
	ctx context.Context,
	id string,
	olds PackageState,
	news PackageArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Package.Diff", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
	))
	defer span.End()

	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: true,
	}

	if news.Name != nil {
		if olds.Name == nil {
			diff.HasChanges = true
			diff.DetailedDiff["name"] = p.PropertyDiff{
				Kind:      p.Add,
				InputDiff: true,
			}
		} else if *news.Name != *olds.Name {
			diff.HasChanges = true
			diff.DetailedDiff["name"] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
	}

	if news.Names != nil {
		if olds.Names == nil {
			diff.HasChanges = true
			diff.DetailedDiff["names"] = p.PropertyDiff{
				Kind:      p.Add,
				InputDiff: true,
			}
		} else if !slices.Equal(*olds.Names, *news.Names) {
			diff.HasChanges = true
			diff.DetailedDiff["names"] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
	} else if olds.Names != nil && !slices.Equal(*olds.Names, *news.Names) {
		diff.HasChanges = true
		diff.DetailedDiff["names"] = p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: true,
		}
	}

	if news.Ensure != nil && *news.Ensure != olds.Ensure {
		diff.HasChanges = true
		diff.DetailedDiff["ensure"] = p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: true,
		}
	}

	diff = types.MergeDiffResponses(diff, types.DiffTriggers(olds, news))

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r Package) Create(
	ctx context.Context,
	name string,
	input PackageArgs,
	preview bool,
) (string, PackageState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Package.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(PackageState{}, input, true)

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

	_, err = executor.AnsibleExecute[
		ansible.PackageParameters,
		ansible.PackageReturn,
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

func (r Package) Read(
	ctx context.Context,
	id string,
	inputs PackageArgs,
	state PackageState,
) (string, PackageArgs, PackageState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Package.Read", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	result, err := executor.AnsibleExecute[
		ansible.PackageParameters,
		ansible.PackageReturn,
	](ctx, config.Connection, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return id, inputs, PackageState{}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	if result.IsChanged() {
		if *inputs.Ensure == "absent" {
			// we're going from present? to absent
			if state.Ensure == "absent" {
				state.Ensure = "present"
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r Package) Update(
	ctx context.Context,
	id string,
	olds PackageState,
	news PackageArgs,
	preview bool,
) (PackageState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Package.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if news.Ensure != nil && *news.Ensure == "absent" {
		parameters, err := r.argsToTaskParameters(news)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return olds, err
		}

		result, err := executor.AnsibleExecute[
			ansible.PackageParameters,
			ansible.PackageReturn,
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

	packageStateMap := map[string]string{}

	newState := olds.Ensure
	if news.Ensure != nil {
		newState = *news.Ensure
	}

	if news.Name != nil {
		packageStateMap[*news.Name] = newState
	} else if news.Names != nil {
		for _, name := range *news.Names {
			packageStateMap[name] = newState
		}
	} else if olds.Name != nil {
		packageStateMap[*olds.Name] = newState
	} else if olds.Names != nil {
		for _, name := range *olds.Names {
			packageStateMap[name] = newState
		}
	} else {
		err := errors.New("we somehow forgot the package name, oops")
		span.SetStatus(codes.Error, err.Error())
		return PackageState{}, err
	}

	if olds.Name != nil {
		if _, exists := packageStateMap[*olds.Name]; !exists {
			packageStateMap[*olds.Name] = "absent"
		}
	} else {
		for _, name := range *olds.Names {
			if _, exists := packageStateMap[name]; !exists {
				packageStateMap[name] = "absent"
			}
		}
	}

	absents := []string{}
	presents := []string{}

	for name, state := range packageStateMap {
		if state == "absent" {
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
		if result.IsChanged() {
			changed = true
		}
	}

	state := r.updateState(olds, news, changed)
	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Package) Delete(ctx context.Context, id string, props PackageState) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:Package.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	if props.Ensure == "absent" {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(PackageArgs{
		Name:   props.Name,
		Names:  props.Names,
		Ensure: ptr.Of("absent"),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = executor.AnsibleExecute[
		ansible.PackageParameters,
		ansible.PackageReturn,
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
