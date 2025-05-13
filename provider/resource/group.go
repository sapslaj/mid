package resource

import (
	"context"
	"errors"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
	"github.com/sapslaj/mid/ptr"
)

type Group struct{}

type GroupArgs struct {
	Name      *string              `pulumi:"name,optional"`
	Ensure    *string              `pulumi:"ensure,optional"`
	Force     *bool                `pulumi:"force,optional"`
	Gid       *int                 `pulumi:"gid,optional"`
	GidMax    *int                 `pulumi:"gidMax,optional"`
	GidMin    *int                 `pulumi:"gidMin,optional"`
	Local     *bool                `pulumi:"local,optional"`
	NonUnique *bool                `pulumi:"nonUnique,optional"`
	System    *bool                `pulumi:"system,optional"`
	Triggers  *types.TriggersInput `pulumi:"triggers,optional"`
}

type GroupState struct {
	GroupArgs
	Name     string               `pulumi:"name"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

type groupTaskParameters struct {
	Force     *bool   `json:"force,omitempty"`
	Gid       *int    `json:"gid,omitempty"`
	GidMax    *int    `json:"gid_max,omitempty"`
	GidMin    *int    `json:"gid_min,omitempty"`
	Local     *bool   `json:"local,omitempty"`
	Name      string  `json:"name"`
	NonUnique *bool   `json:"non_unique,omitempty"`
	State     *string `json:"state,omitempty"`
	System    *bool   `json:"system,omitempty"`
}

type groupTaskResult struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (result *groupTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r Group) argsToTaskParameters(input GroupArgs) (groupTaskParameters, error) {
	if input.Name == nil {
		return groupTaskParameters{}, errors.New("someone forgot to set the auto-named input.Name")
	}
	return groupTaskParameters{
		Force:     input.Force,
		Gid:       input.Gid,
		GidMax:    input.GidMax,
		GidMin:    input.GidMin,
		Local:     input.Local,
		Name:      *input.Name,
		NonUnique: input.NonUnique,
		State:     input.Ensure,
		System:    input.System,
	}, nil
}

func (r Group) updateState(olds GroupState, news GroupArgs, changed bool) GroupState {
	olds.GroupArgs = news
	if news.Name != nil {
		olds.Name = *news.Name
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Group) Diff(
	ctx context.Context,
	id string,
	olds GroupState,
	news GroupArgs,
) (p.DiffResponse, error) {
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
			"ensure",
			"force",
			"gid",
			"gidMax",
			"gidMin",
			"local",
			"nonUnique",
			"system",
		}),
		types.DiffTriggers(olds, news),
	)

	return diff, nil
}

func (r Group) Create(
	ctx context.Context,
	name string,
	input GroupArgs,
	preview bool,
) (string, GroupState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(GroupState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		return id, state, err
	}

	connectAttempts := 10
	if preview {
		connectAttempts = 4
	}
	canConnect, err := executor.CanConnect(ctx, config.Connection, connectAttempts)

	if !canConnect {
		if preview {
			return id, state, nil
		}

		if err == nil {
			return id, state, fmt.Errorf("cannot connect to host")
		} else {
			return id, state, fmt.Errorf("cannot connect to host: %w", err)
		}
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.group": parameters,
			},
		},
	})
	if err != nil {
		return id, state, err
	}

	return id, state, nil
}

func (r Group) Read(
	ctx context.Context,
	id string,
	inputs GroupArgs,
	state GroupState,
) (string, GroupArgs, GroupState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil {
		inputs.Name = ptr.Of(state.Name)
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		return id, inputs, state, err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection, 4)

	if !canConnect {
		return id, inputs, GroupState{
			GroupArgs: inputs,
		}, nil
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       true,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.group": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*groupTaskResult](output, 0, 0)
	if err != nil {
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	return id, inputs, state, nil
}

func (r Group) Update(
	ctx context.Context,
	id string,
	olds GroupState,
	news GroupArgs,
	preview bool,
) (GroupState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil {
		news.Name = ptr.Of(olds.Name)
	}

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		return olds, err
	}

	connectAttempts := 10
	if preview {
		connectAttempts = 4
	}
	canConnect, err := executor.CanConnect(ctx, config.Connection, connectAttempts)

	if !canConnect {
		if preview {
			return olds, nil
		}

		if err == nil {
			return olds, fmt.Errorf("cannot connect to host")
		} else {
			return olds, fmt.Errorf("cannot connect to host: %w", err)
		}
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.group": parameters,
			},
		},
	})
	if err != nil {
		return olds, err
	}

	result, err := executor.GetTaskResult[*groupTaskResult](output, 0, 0)

	state := r.updateState(olds, news, result.IsChanged())
	return state, nil
}

func (r Group) Delete(
	ctx context.Context,
	id string,
	props GroupState,
) error {
	if props.Ensure != nil && *props.Ensure == "absent" {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	args := props.GroupArgs
	args.Name = &props.Name

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		return err
	}
	parameters.State = ptr.Of("absent")

	canConnect, err := executor.CanConnect(ctx, config.Connection, 10)

	if !canConnect {
		if config.GetDeleteUnreachable() {
			return nil
		}

		if err == nil {
			return fmt.Errorf("cannot connect to host")
		} else {
			return fmt.Errorf("cannot connect to host: %w", err)
		}
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.group": parameters,
			},
		},
	})
	return err
}
