package resource

import (
	"context"
	"errors"
	"slices"

	"github.com/aws/smithy-go/ptr"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

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

type packageTaskParameters struct {
	Name  any    `json:"name"`
	State string `json:"state"`
}

type packageTaskResult struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (result *packageTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r Package) argsToTaskParameters(input PackageArgs) (packageTaskParameters, error) {
	parameters := packageTaskParameters{}
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

	return diff, nil
}

func (r Package) Create(
	ctx context.Context,
	name string,
	input PackageArgs,
	preview bool,
) (string, PackageState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil && input.Names == nil {
		input.Name = ptr.String(name)
	}

	state := r.updateState(PackageState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		return id, state, err
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.package": parameters,
				"ignore_errors":           preview,
			},
		},
	})
	if err != nil {
		return id, state, err
	}

	return id, state, nil
}

func (r Package) Read(
	ctx context.Context,
	id string,
	inputs PackageArgs,
	state PackageState,
) (string, PackageArgs, PackageState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil && inputs.Names == nil && state.Name != nil {
		inputs.Name = state.Name
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		return id, inputs, state, err
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       true,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.package": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*packageTaskResult](output, 0, 0)
	if err != nil {
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

	return id, inputs, state, nil
}

func (r Package) Update(
	ctx context.Context,
	id string,
	olds PackageState,
	news PackageArgs,
	preview bool,
) (PackageState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil && news.Names == nil && olds.Name != nil {
		news.Name = olds.Name
	}

	if news.Ensure != nil && *news.Ensure == "absent" {
		parameters, err := r.argsToTaskParameters(news)
		if err != nil {
			return olds, err
		}

		output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
			GatherFacts: true,
			Become:      true,
			Check:       preview,
			Tasks: []any{
				map[string]any{
					"ansible.builtin.package": parameters,
					"ignore_errors":           preview,
				},
			},
		})
		if err != nil {
			return olds, err
		}

		result, err := executor.GetTaskResult[*packageTaskResult](output, 0, 0)
		if err != nil {
			return olds, err
		}

		state := r.updateState(olds, news, result.IsChanged())

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
		return PackageState{}, errors.New("we somehow forgot the package name, oops")
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

	taskParameterSets := []packageTaskParameters{}

	absents := []string{}
	presents := []string{}

	for name, state := range packageStateMap {
		if state == "absent" {
			absents = append(absents, name)
		} else {
			presents = append(presents, name)
		}
	}

	if len(absents) > 0 {
		taskParameterSets = append(taskParameterSets, packageTaskParameters{
			Name:  absents,
			State: "absent",
		})
	}

	if len(presents) > 0 {
		taskParameterSets = append(taskParameterSets, packageTaskParameters{
			Name:  presents,
			State: newState,
		})
	}

	if len(taskParameterSets) == 0 {
		return olds, errors.New("could not figure out how to update this thing")
	}

	tasks := []any{}
	for _, parameters := range taskParameterSets {
		tasks = append(tasks, map[string]any{
			"ansible.builtin.package": parameters,
			"ignore_errors":           preview,
		})
	}
	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       preview,
		Tasks:       tasks,
	})
	if err != nil {
		return olds, err
	}

	changed := false
	for i := range output.Results[0].Tasks {
		r, err := executor.GetTaskResult[*packageTaskResult](output, 0, i)
		if err != nil {
			return olds, err
		}
		if r.IsChanged() {
			changed = true
			break
		}
	}

	state := r.updateState(olds, news, changed)
	return state, nil
}

func (r Package) Delete(ctx context.Context, id string, props PackageState) error {
	if props.Ensure == "absent" {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(PackageArgs{
		Name:   props.Name,
		Names:  props.Names,
		Ensure: ptr.String("absent"),
	})
	if err != nil {
		return err
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.package": parameters,
			},
		},
	})

	return err
}
