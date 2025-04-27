package resource

import (
	"context"
	"reflect"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
)

type Exec struct{}

type ExecArgs struct {
	Create             types.ExecCommand    `pulumi:"create"`
	Update             *types.ExecCommand   `pulumi:"update,optional"`
	Delete             *types.ExecCommand   `pulumi:"delete,optional"`
	ExpandArgumentVars *bool                `pulumi:"expandArgumentVars,optional"`
	Dir                *string              `pulumi:"dir,optional"`
	Environment        *map[string]string   `pulumi:"environment,optional"`
	Logging            *types.ExecLogging   `pulumi:"logging,optional"`
	Triggers           *types.TriggersInput `pulumi:"triggers,optional"`
}

type ExecState struct {
	ExecArgs
	Stdout   string               `pulumi:"stdout"`
	Stderr   string               `pulumi:"stderr"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

type commandTaskParameters struct {
	Argv               []string `json:"argv"`
	Chdir              *string  `json:"chdir,omitempty"`
	ExpandArgumentVars bool     `json:"expand_argument_vars"`
	Stdin              *string  `json:"stdin,omitempty"`
}

type commandTaskResult struct {
	// TODO: pluck stdout and stderr from here
}

func (r Exec) argsToTaskParameters(input ExecArgs, lifecycle string) (commandTaskParameters, error) {
	var execCommand types.ExecCommand
	switch lifecycle {
	case "create":
		execCommand = input.Create
	case "update":
		if input.Update != nil {
			execCommand = *input.Update
		} else {
			execCommand = input.Create
		}
	case "delete":
		if input.Delete == nil {
			return commandTaskParameters{}, nil
		}
		execCommand = *input.Delete
	default:
		panic("unknown lifecycle: " + lifecycle)
	}

	chdir := input.Dir
	if execCommand.Dir != nil {
		chdir = execCommand.Dir
	}
	expandArgumentVars := false
	if input.ExpandArgumentVars != nil {
		expandArgumentVars = *input.ExpandArgumentVars
	}

	return commandTaskParameters{
		Argv:               execCommand.Command,
		Chdir:              chdir,
		Stdin:              execCommand.Stdin,
		ExpandArgumentVars: expandArgumentVars,
	}, nil
}

func (r Exec) updateState(olds ExecState, news ExecArgs, changed bool) ExecState {
	olds.ExecArgs = news
	if news.Triggers != nil {
		olds.Triggers.Replace = news.Triggers.Replace
		olds.Triggers.Refresh = news.Triggers.Refresh
	}
	if changed {
		olds.Triggers.LastChanged = time.Now().UTC().Format(time.RFC3339)
	}
	return olds
}

func (r Exec) Diff(
	ctx context.Context,
	id string,
	olds ExecState,
	news ExecArgs,
) (p.DiffResponse, error) {
	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	for _, pair := range [][]any{
		{"create", olds.Create, news.Create},
		{"update", olds.Update, news.Update},
		{"delete", olds.Delete, news.Delete},
		{"expandArgumentVars", olds.ExpandArgumentVars, news.ExpandArgumentVars},
		{"dir", olds.Dir, news.Dir},
		{"environment", olds.Environment, news.Environment},
		{"loggin", olds.Logging, news.Logging},
	} {
		key := pair[0].(string)
		o := pair[1]
		n := pair[2]

		if reflect.ValueOf(n).IsNil() {
			continue
		}

		if reflect.ValueOf(o).IsNil() {
			diff.HasChanges = true
			diff.DetailedDiff[key] = p.PropertyDiff{
				Kind:      p.Add,
				InputDiff: true,
			}
			continue
		}

		if !resource.NewPropertyValue(o).DeepEquals(resource.NewPropertyValue(n)) {
			diff.HasChanges = true
			diff.DetailedDiff[key] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
	}

	if news.Triggers != nil {
		refreshDiff := resource.NewPropertyValue(olds.Triggers.Refresh).Diff(resource.NewPropertyValue(news.Triggers.Refresh))
		if refreshDiff != nil {
			diff.HasChanges = true
			diff.DetailedDiff["triggers"] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
		replaceDiff := resource.NewPropertyValue(olds.Triggers.Replace).Diff(resource.NewPropertyValue(news.Triggers.Replace))
		if replaceDiff != nil {
			diff.HasChanges = true
			diff.DetailedDiff["triggers"] = p.PropertyDiff{
				Kind:      p.UpdateReplace,
				InputDiff: true,
			}
		}
	}

	return diff, nil
}

func (r Exec) Create(
	ctx context.Context,
	name string,
	input ExecArgs,
	preview bool,
) (string, ExecState, error) {
	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(ExecState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input, "create")
	if err != nil {
		return id, state, err
	}

	if !preview {
		// TODO: slurp out stdout and stderr
		_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
			GatherFacts: false,
			Become:      true,
			Check:       false,
			Tasks: []any{
				map[string]any{
					"ansible.builtin.command": parameters,
				},
			},
		})
		if err != nil {
			return id, state, err
		}
	}

	return id, state, nil
}

// func (r Exec) Read(
// 	ctx context.Context,
// 	id string,
// 	inputs ExecArgs,
// 	state ExecState,
// ) (string, ExecArgs, ExecState, error) {
// 	config := infer.GetConfig[types.Config](ctx)
// }

func (r Exec) Update(
	ctx context.Context,
	id string,
	olds ExecState,
	news ExecArgs,
	preview bool,
) (ExecState, error) {
	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(news, "create")
	if err != nil {
		return olds, err
	}

	if !preview {
		// TODO: slurp out stdout and stderr
		_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
			GatherFacts: false,
			Become:      true,
			Check:       false,
			Tasks: []any{
				map[string]any{
					"ansible.builtin.command": parameters,
				},
			},
		})
	}
	state := r.updateState(olds, news, true)
	if err != nil {
		return state, err
	}

	return state, nil
}

func (r Exec) Delete(ctx context.Context, id string, props ExecState) error {
	if props.Delete == nil {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)
	parameters, err := r.argsToTaskParameters(props.ExecArgs, "delete")
	if err != nil {
		return err
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.command": parameters,
			},
		},
	})

	return err
}
