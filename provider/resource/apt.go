package resource

import (
	"context"
	"errors"
	"slices"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
	"github.com/sapslaj/mid/ptr"
)

type Apt struct{}

type AptArgs struct {
	Name                     *string              `pulumi:"name,optional"`
	Names                    *[]string            `pulumi:"names,optional"`
	Ensure                   *string              `pulumi:"ensure,optional"`
	AllowChangeHeldPackages  *bool                `pulumi:"allowChangeHeldPackages,optional"`
	AllowDowngrade           *bool                `pulumi:"allowDowngrade,optional"`
	AllowUnauthenticated     *bool                `pulumi:"allowUnauthenticated,optional"`
	Autoclean                *bool                `pulumi:"autoclean,optional"`
	Autoremove               *bool                `pulumi:"autoremove,optional"`
	CacheValidTime           *int                 `pulumi:"cacheValidTime,optional"`
	Clean                    *bool                `pulumi:"clean,optional"`
	Deb                      *string              `pulumi:"deb,optional"`
	DefaultRelease           *string              `pulumi:"defaultRelease,optional"`
	DpkgOptions              *string              `pulumi:"dpkgOptions,optional"`
	FailOnAutoremove         *bool                `pulumi:"failOnAutoremove,optional"`
	Force                    *bool                `pulumi:"force,optional"`
	ForceAptGet              *bool                `pulumi:"forceAptGet,optional"`
	InstallRecommends        *bool                `pulumi:"installRecommends,optional"`
	LockTimeout              *int                 `pulumi:"lockTimeout,optional"`
	OnlyUpgrade              *bool                `pulumi:"onlyUpgrade,optional"`
	PolicyRcD                *int                 `pulumi:"policyRcD,optional"`
	Purge                    *bool                `pulumi:"purge,optional"`
	UpdateCache              *bool                `pulumi:"updateCache,optional"`
	UpdateCacheRetries       *int                 `pulumi:"updateCacheRetries,optional"`
	UpdateCacheRetryMaxDelay *int                 `pulumi:"updateCacheRetryMaxDelay,optional"`
	Upgrade                  *string              `pulumi:"upgrade,optional"`
	Triggers                 *types.TriggersInput `pulumi:"triggers,optional"`
}

type AptState struct {
	AptArgs
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

type aptTaskParameters struct {
	AllowChangeHeldPackages  *bool     `json:"allow_change_held_packages,omitempty"`
	AllowDowngrade           *bool     `json:"allow_downgrade,omitempty"`
	AllowUnauthenticated     *bool     `json:"allow_unauthenticated,omitempty"`
	Autoclean                *bool     `json:"autoclean,omitempty"`
	Autoremove               *bool     `json:"autoremove,omitempty"`
	CacheValidTime           *int      `json:"cache_valid_time,omitempty"`
	Clean                    *bool     `json:"clean,omitempty"`
	Deb                      *string   `json:"deb,omitempty"`
	DefaultRelease           *string   `json:"default_release,omitempty"`
	DpkgOptions              *string   `json:"dpkg_options,omitempty"`
	FailOnAutoremove         *bool     `json:"fail_on_autoremove,omitempty"`
	Force                    *bool     `json:"force,omitempty"`
	ForceAptGet              *bool     `json:"force_apt_get,omitempty"`
	InstallRecommends        *bool     `json:"install_recommends,omitempty"`
	LockTimeout              *int      `json:"lock_timeout,omitempty"`
	Name                     *[]string `json:"name,omitempty"`
	OnlyUpgrade              *bool     `json:"only_upgrade,omitempty"`
	PolicyRcD                *int      `json:"policy_rc_d,omitempty"`
	Purge                    *bool     `json:"purge,omitempty"`
	State                    *string   `json:"state,omitempty"`
	UpdateCache              *bool     `json:"update_cache,omitempty"`
	UpdateCacheRetries       *int      `json:"update_cache_retries,omitempty"`
	UpdateCacheRetryMaxDelay *int      `json:"update_cache_retry_max_delay,omitempty"`
	Upgrade                  *string   `json:"upgrade,omitempty"`
}

type aptTaskResult struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (result *aptTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r Apt) taskParametersNeedsName(input AptArgs) bool {
	return !anyNonNils(
		input.Autoclean,
		input.Autoremove,
		input.Clean,
		input.Deb,
		input.UpdateCache,
		input.Upgrade,
	)
}

func (r Apt) canAssumeEnsure(input AptArgs) bool {
	if anyNonNils(
		input.Name,
		input.Names,
		input.Deb,
	) {
		return true
	}

	return r.taskParametersNeedsName(input)
}

func (r Apt) argsToTaskParameters(input AptArgs) (aptTaskParameters, error) {
	parameters := aptTaskParameters{
		AllowChangeHeldPackages:  input.AllowChangeHeldPackages,
		AllowDowngrade:           input.AllowDowngrade,
		AllowUnauthenticated:     input.AllowUnauthenticated,
		Autoclean:                input.Autoclean,
		Autoremove:               input.Autoremove,
		CacheValidTime:           input.CacheValidTime,
		Clean:                    input.Clean,
		Deb:                      input.Deb,
		DefaultRelease:           input.DefaultRelease,
		DpkgOptions:              input.DpkgOptions,
		FailOnAutoremove:         input.FailOnAutoremove,
		Force:                    input.Force,
		ForceAptGet:              input.ForceAptGet,
		InstallRecommends:        input.InstallRecommends,
		LockTimeout:              input.LockTimeout,
		Name:                     input.Names,
		OnlyUpgrade:              input.OnlyUpgrade,
		PolicyRcD:                input.PolicyRcD,
		Purge:                    input.Purge,
		State:                    input.Ensure,
		UpdateCache:              input.UpdateCache,
		UpdateCacheRetries:       input.UpdateCacheRetries,
		UpdateCacheRetryMaxDelay: input.UpdateCacheRetryMaxDelay,
		Upgrade:                  input.Upgrade,
	}

	if input.Name != nil && parameters.Name == nil {
		parameters.Name = ptr.Of([]string{*input.Name})
	}

	return parameters, nil
}

func (r Apt) updateState(olds AptState, news AptArgs, changed bool) AptState {
	olds.AptArgs = news
	if olds.Ensure == nil && r.canAssumeEnsure(news) {
		olds.Ensure = ptr.Of("present")
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Apt) Diff(
	ctx context.Context,
	id string,
	olds AptState,
	news AptArgs,
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

	attrs := []string{
		"allowChangeHeldPackages",
		"allowDowngrade",
		"allowUnauthenticated",
		"autoclean",
		"autoremove",
		"cacheValidTime",
		"clean",
		"deb",
		"defaultRelease",
		"dpkgOptions",
		"failOnAutoremove",
		"force",
		"forceAptGet",
		"installRecommends",
		"lockTimeout",
		"onlyUpgrade",
		"policyRcD",
		"purge",
		"updateCache",
		"updateCacheRetries",
		"updateCacheRetryMaxDelay",
		"upgrade",
	}
	if news.Ensure == nil && r.canAssumeEnsure(news) && olds.Ensure != nil {
		// special diff for "ensure" since we compute it dynamically sometimes
		pdiff := types.DiffAttribute(olds.Ensure, news.Ensure)
		if pdiff != nil {
			diff.HasChanges = true
			diff.DetailedDiff["ensure"] = *pdiff
		}
	} else {
		// just do standard diffing
		attrs = append(attrs, "ensure")
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, attrs),
		types.DiffTriggers(olds, news),
	)

	return diff, nil
}

func (r Apt) Create(
	ctx context.Context,
	name string,
	input AptArgs,
	preview bool,
) (string, AptState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if r.taskParametersNeedsName(input) && input.Name == nil && input.Names == nil {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(AptState{}, input, true)

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
				"ansible.builtin.apt": parameters,
				"ignore_errors":       preview,
			},
		},
	})
	if err != nil {
		return id, state, err
	}

	return id, state, nil
}

func (r Apt) Read(
	ctx context.Context,
	id string,
	inputs AptArgs,
	state AptState,
) (string, AptArgs, AptState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if r.taskParametersNeedsName(inputs) && inputs.Name == nil && inputs.Names == nil && state.Name != nil {
		inputs.Name = state.Name
	}

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
				"ansible.builtin.apt": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*aptTaskResult](output, 0, 0)
	if err != nil {
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	if result.IsChanged() && r.canAssumeEnsure(inputs) {
		if inputs.Ensure != nil && *inputs.Ensure == "absent" {
			if state.Ensure == nil || *state.Ensure == "absent" {
				state.Ensure = ptr.Of("present")
			}
		}
	}

	return id, inputs, state, nil
}

func (r Apt) Update(
	ctx context.Context,
	id string,
	olds AptState,
	news AptArgs,
	preview bool,
) (AptState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if r.taskParametersNeedsName(news) && news.Name == nil && news.Names == nil && olds.Name != nil {
		news.Name = olds.Name
	}

	if (news.Ensure != nil && *news.Ensure == "absent") || !r.canAssumeEnsure(news) {
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
					"ansible.builtin.apt": parameters,
					"ignore_errors":       preview,
				},
			},
		})
		if err != nil {
			return olds, err
		}

		result, err := executor.GetTaskResult[*aptTaskResult](output, 0, 0)
		if err != nil {
			return olds, err
		}

		state := r.updateState(olds, news, result.IsChanged())

		return state, nil
	}

	aptStateMap := map[string]string{}

	newState := "present"
	if olds.Ensure != nil {
		newState = *olds.Ensure
	}
	if news.Ensure != nil {
		newState = *news.Ensure
	}

	if news.Name != nil {
		aptStateMap[*news.Name] = newState
	} else if news.Names != nil {
		for _, name := range *news.Names {
			aptStateMap[name] = newState
		}
	} else if olds.Name != nil {
		aptStateMap[*olds.Name] = newState
	} else if olds.Names != nil {
		for _, name := range *olds.Names {
			aptStateMap[name] = newState
		}
	} else {
		return AptState{}, errors.New("we somehow forgot the apt name, oops")
	}

	if olds.Name != nil {
		if _, exists := aptStateMap[*olds.Name]; !exists {
			aptStateMap[*olds.Name] = "absent"
		}
	} else {
		for _, name := range *olds.Names {
			if _, exists := aptStateMap[name]; !exists {
				aptStateMap[name] = "absent"
			}
		}
	}

	taskParameterSets := []aptTaskParameters{}

	absents := []string{}
	presents := []string{}

	for name, state := range aptStateMap {
		if state == "absent" {
			absents = append(absents, name)
		} else {
			presents = append(presents, name)
		}
	}

	if len(absents) > 0 {
		taskParameterSets = append(taskParameterSets, aptTaskParameters{
			Name:  ptr.Of(absents),
			State: ptr.Of("absent"),
		})
	}

	if len(presents) > 0 {
		taskParameterSets = append(taskParameterSets, aptTaskParameters{
			Name:  ptr.Of(presents),
			State: ptr.Of(newState),
		})
	}

	if len(taskParameterSets) == 0 {
		return olds, errors.New("could not figure out how to update this thing")
	}

	tasks := []any{}
	for _, parameters := range taskParameterSets {
		tasks = append(tasks, map[string]any{
			"ansible.builtin.apt": parameters,
			"ignore_errors":       preview,
		})
	}
	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       preview,
		Tasks:       tasks,
	})
	if err != nil {
		return olds, err
	}

	changed := false
	for i := range output.Results[0].Tasks {
		r, err := executor.GetTaskResult[*aptTaskResult](output, 0, i)
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

func (r Apt) Delete(ctx context.Context, id string, props AptState) error {
	if !r.taskParametersNeedsName(props.AptArgs) {
		return nil
	}

	if props.Ensure != nil && *props.Ensure == "absent" {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(props.AptArgs)
	parameters.State = ptr.Of("absent")
	if err != nil {
		return err
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.apt": parameters,
			},
		},
	})

	return err
}
