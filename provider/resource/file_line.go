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

type FileLine struct{}

type FileLineArgs struct {
	Ensure       *string              `pulumi:"ensure,optional"`
	Path         string               `pulumi:"path"`
	Backrefs     *bool                `pulumi:"backrefs,optional"`
	Backup       *bool                `pulumi:"backup,optional"`
	Create       *bool                `pulumi:"create,optional"`
	FirstMatch   *bool                `pulumi:"firstMatch,optional"`
	InsertBefore *string              `pulumi:"insertBefore,optional"`
	InsertAfter  *string              `pulumi:"insertAfter,optional"`
	Line         *string              `pulumi:"line,optional"`
	Regexp       *string              `pulumi:"regexp,optional"`
	SearchString *string              `pulumi:"searchString,optional"`
	UnsafeWrites *bool                `pulumi:"unsafeWrites,optional"`
	Validate     *string              `pulumi:"validate,optional"`
	Triggers     *types.TriggersInput `pulumi:"triggers,optional"`
}

type FileLineState struct {
	FileLineArgs
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

type lineinfileTaskParameters struct {
	Attributes   *string `json:"attributes,omitempty"`
	Backrefs     *bool   `json:"backrefs,omitempty"`
	Backup       *bool   `json:"backup,omitempty"`
	Create       *bool   `json:"create,omitempty"`
	Firstmatch   *bool   `json:"firstmatch,omitempty"`
	Group        *string `json:"group,omitempty"`
	Insertafter  *string `json:"insertafter,omitempty"`
	Insertbefore *string `json:"insertbefore,omitempty"`
	Line         *string `json:"line,omitempty"`
	Mode         any     `json:"mode,omitempty"`
	Owner        *string `json:"owner,omitempty"`
	Path         string  `json:"path"`
	Regexp       *string `json:"regexp,omitempty"`
	SearchString *string `json:"search_string,omitempty"`
	Selevel      *string `json:"selevel,omitempty"`
	Serole       *string `json:"serole,omitempty"`
	Setype       *string `json:"setype,omitempty"`
	Seuser       *string `json:"seuser,omitempty"`
	State        *string `json:"state,omitempty"`
	UnsafeWrites *bool   `json:"unsafe_writes,omitempty"`
	Validate     *string `json:"validate,omitempty"`
}

type lineinfileTaskResult struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (result *lineinfileTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r FileLine) argsToTaskParameters(input FileLineArgs) (lineinfileTaskParameters, error) {
	return lineinfileTaskParameters{
		State:        input.Ensure,
		Path:         input.Path,
		Backrefs:     input.Backrefs,
		Backup:       input.Backup,
		Create:       input.Create,
		Firstmatch:   input.FirstMatch,
		Insertbefore: input.InsertBefore,
		Insertafter:  input.InsertAfter,
		Line:         input.Line,
		Regexp:       input.Regexp,
		SearchString: input.SearchString,
		UnsafeWrites: input.UnsafeWrites,
		Validate:     input.Validate,
	}, nil
}

func (r FileLine) updateState(olds FileLineState, news FileLineArgs, changed bool) FileLineState {
	olds.FileLineArgs = news
	if news.Triggers != nil {
		olds.Triggers.Replace = news.Triggers.Replace
		olds.Triggers.Refresh = news.Triggers.Refresh
	}
	if changed {
		olds.Triggers.LastChanged = time.Now().UTC().Format(time.RFC3339)
	}
	return olds
}

func (r FileLine) Diff(
	ctx context.Context,
	id string,
	olds FileLineState,
	news FileLineArgs,
) (p.DiffResponse, error) {
	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	for _, pair := range [][]any{
		{"ensure", olds.Ensure, news.Ensure},
		{"path", olds.Path, news.Path},
		{"backrefs", olds.Backrefs, news.Backrefs},
		{"backup", olds.Backup, news.Backup},
		{"create", olds.Create, news.Create},
		{"firstMatch", olds.FirstMatch, news.FirstMatch},
		{"insertBefore", olds.InsertBefore, news.InsertBefore},
		{"insertAfter", olds.InsertAfter, news.InsertAfter},
		{"line", olds.Line, news.Line},
		{"regexp", olds.Regexp, news.Regexp},
		{"searchString", olds.SearchString, news.SearchString},
		{"unsafeWrites", olds.UnsafeWrites, news.UnsafeWrites},
		{"validate", olds.Validate, news.Validate},
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

func (r FileLine) Create(
	ctx context.Context,
	name string,
	input FileLineArgs,
	preview bool,
) (string, FileLineState, error) {
	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(FileLineState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		return id, state, err
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.lineinfile": parameters,
			},
		},
	})
	if err != nil {
		return id, state, err
	}

	return id, state, nil
}

func (r FileLine) Read(
	ctx context.Context,
	id string,
	inputs FileLineArgs,
	state FileLineState,
) (string, FileLineArgs, FileLineState, error) {
	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		return id, inputs, state, err
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       true,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.lineinfile": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*lineinfileTaskResult](output, 0, 0)
	if err != nil {
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	return id, inputs, state, nil
}

func (r FileLine) Update(
	ctx context.Context,
	id string,
	olds FileLineState,
	news FileLineArgs,
	preview bool,
) (FileLineState, error) {
	config := infer.GetConfig[types.Config](ctx)

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
				"ansible.builtin.lineinfile": parameters,
			},
		},
	})
	if err != nil {
		return olds, err
	}

	result, err := executor.GetTaskResult[*lineinfileTaskResult](output, 0, 0)

	state := r.updateState(olds, news, result.IsChanged())
	return state, nil
}
