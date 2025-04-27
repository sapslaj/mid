package resource

import (
	"context"

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
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
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
		DeleteBeforeReplace: true,
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
			"ensure",
			"path",
			"backrefs",
			"backup",
			"create",
			"firstMatch",
			"insertBefore",
			"insertAfter",
			"line",
			"regexp",
			"searchString",
			"unsafeWrites",
			"validate",
		}),
		types.DiffTriggers(olds, news),
	)

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
		return olds, err
	}

	result, err := executor.GetTaskResult[*lineinfileTaskResult](output, 0, 0)

	state := r.updateState(olds, news, result.IsChanged())
	return state, nil
}
