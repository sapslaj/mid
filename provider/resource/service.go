package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/smithy-go/ptr"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

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

type serviceTaskParameters struct {
	Arguments *string `json:"arguments,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Name      string  `json:"name"`
	Pattern   *string `json:"pattern,omitempty"`
	Runlevel  *string `json:"runlevel,omitempty"`
	Sleep     *int    `json:"sleep,omitempty"`
	State     *string `json:"state,omitempty"` // TODO: enum for this?
	Use       *string `json:"use,omitempty"`
}

type serviceTaskResult struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (result *serviceTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r Service) argsToTaskParameters(input ServiceArgs) (serviceTaskParameters, error) {
	if input.Name == nil {
		return serviceTaskParameters{}, errors.New("someone forgot to set the auto-named input.Name")
	}
	return serviceTaskParameters{
		Arguments: input.Arguments,
		Enabled:   input.Enabled,
		Name:      *input.Name,
		Pattern:   input.Pattern,
		Runlevel:  input.Runlevel,
		Sleep:     input.Sleep,
		State:     input.State,
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

	return diff, nil
}

func (r Service) Create(
	ctx context.Context,
	name string,
	input ServiceArgs,
	preview bool,
) (string, ServiceState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil {
		input.Name = ptr.String(name)
	}

	state := r.updateState(ServiceState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		return id, state, err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection)

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
				"ansible.builtin.service": parameters,
				"ignore_errors":           preview,
			},
		},
	})
	if err != nil {
		return id, state, err
	}

	return id, state, nil
}

func (r Service) Read(
	ctx context.Context,
	id string,
	inputs ServiceArgs,
	state ServiceState,
) (string, ServiceArgs, ServiceState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil {
		inputs.Name = ptr.String(state.Name)
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		return id, inputs, state, err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection)

	if !canConnect {
		return id, inputs, ServiceState{
			ServiceArgs: inputs,
		}, nil
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       true,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.service": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*serviceTaskResult](output, 0, 0)
	if err != nil {
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	return id, inputs, state, nil
}

func (r Service) Update(
	ctx context.Context,
	id string,
	olds ServiceState,
	news ServiceArgs,
	preview bool,
) (ServiceState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil {
		news.Name = ptr.String(olds.Name)
	}

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		return olds, err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection)

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
				"ansible.builtin.service": parameters,
				"ignore_errors":           preview,
			},
		},
	})
	if err != nil {
		return olds, err
	}

	result, err := executor.GetTaskResult[*serviceTaskResult](output, 0, 0)

	state := r.updateState(olds, news, result.IsChanged())
	return state, nil
}

func (r Service) Delete(
	ctx context.Context,
	id string,
	props ServiceState,
) error {
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
		args.Enabled = ptr.Bool(false)
	}
	if args.State != nil && *args.State != "stopped" {
		runPlay = true
		args.State = ptr.String("stopped")
	}

	if !runPlay {
		return nil
	}

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		return err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection)

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
				"ansible.builtin.service": parameters,
			},
		},
	})
	return err
}
