package resource

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/aws/smithy-go/ptr"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
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
	AllowChangeHeldPackages  *bool   `json:"allow_change_held_packages,omitempty"`
	AllowDowngrade           *bool   `json:"allow_downgrade,omitempty"`
	AllowUnauthenticated     *bool   `json:"allow_unauthenticated,omitempty"`
	Autoclean                *bool   `json:"autoclean,omitempty"`
	Autoremove               *bool   `json:"autoremove,omitempty"`
	CacheValidTime           *int    `json:"cache_valid_time,omitempty"`
	Clean                    *bool   `json:"clean,omitempty"`
	Deb                      *string `json:"deb,omitempty"`
	DefaultRelease           *string `json:"default_release,omitempty"`
	DpkgOptions              *string `json:"dpkg_options,omitempty"`
	FailOnAutoremove         *bool   `json:"fail_on_autoremove,omitempty"`
	Force                    *bool   `json:"force,omitempty"`
	ForceAptGet              *bool   `json:"force_apt_get,omitempty"`
	InstallRecommends        *bool   `json:"install_recommends,omitempty"`
	LockTimeout              *int    `json:"lock_timeout,omitempty"`
	Name                     any     `json:"name,omitempty"`
	OnlyUpgrade              *bool   `json:"only_upgrade,omitempty"`
	PolicyRcD                *int    `json:"policy_rc_d,omitempty"`
	Purge                    *bool   `json:"purge,omitempty"`
	State                    *string `json:"state,omitempty"`
	UpdateCache              *bool   `json:"update_cache,omitempty"`
	UpdateCacheRetries       *int    `json:"update_cache_retries,omitempty"`
	UpdateCacheRetryMaxDelay *int    `json:"update_cache_retry_max_delay,omitempty"`
	Upgrade                  *string `json:"upgrade,omitempty"`
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
		OnlyUpgrade:              input.OnlyUpgrade,
		PolicyRcD:                input.PolicyRcD,
		Purge:                    input.Purge,
		State:                    input.Ensure,
		UpdateCache:              input.UpdateCache,
		UpdateCacheRetries:       input.UpdateCacheRetries,
		UpdateCacheRetryMaxDelay: input.UpdateCacheRetryMaxDelay,
		Upgrade:                  input.Upgrade,
	}

	if input.Names == nil && input.Name != nil {
		parameters.Name = *input.Name
	} else if input.Names != nil && len(*input.Names) == 1 {
		parameters.Name = (*input.Names)[0]
	} else if input.Names != nil {
		parameters.Name = *input.Names
	}
	if parameters.State == nil && parameters.Name != nil && parameters.Upgrade == nil && parameters.UpdateCache == nil {
		parameters.State = ptr.String("present")
	}
	return parameters, nil
}

func (r Apt) updateState(olds AptState, news AptArgs, changed bool) AptState {
	if news.Name != nil || news.Names != nil {
		olds.Name = news.Name
		olds.Names = news.Names
	}
	if news.Ensure != nil {
		olds.Ensure = news.Ensure
	} else if news.Upgrade == nil && news.UpdateCache == nil {
		olds.Ensure = ptr.String("present")
	}
	if news.Triggers != nil {
		olds.Triggers.Replace = news.Triggers.Replace
		olds.Triggers.Refresh = news.Triggers.Refresh
	}
	if changed {
		olds.Triggers.LastChanged = time.Now().UTC().Format(time.RFC3339)
	}
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

	for _, pair := range [][]any{
		{"ensure", olds.Ensure, news.Ensure},
		{"name", olds.Name, news.Name},
		{"names", olds.Names, news.Names},
		{"allowChangeHeldPackages", olds.AllowChangeHeldPackages, news.AllowChangeHeldPackages},
		{"allowDowngrade", olds.AllowDowngrade, news.AllowDowngrade},
		{"allowUnauthenticated", olds.AllowUnauthenticated, news.AllowUnauthenticated},
		{"autoclean", olds.Autoclean, news.Autoclean},
		{"autoremove", olds.Autoremove, news.Autoremove},
		{"cacheValidTime", olds.CacheValidTime, news.CacheValidTime},
		{"clean", olds.Clean, news.Clean},
		{"deb", olds.Deb, news.Deb},
		{"defaultRelease", olds.DefaultRelease, news.DefaultRelease},
		{"dpkgOptions", olds.DpkgOptions, news.DpkgOptions},
		{"failOnAutoremove", olds.FailOnAutoremove, news.FailOnAutoremove},
		{"force", olds.Force, news.Force},
		{"forceAptGet", olds.ForceAptGet, news.ForceAptGet},
		{"installRecommends", olds.InstallRecommends, news.InstallRecommends},
		{"lockTimeout", olds.LockTimeout, news.LockTimeout},
		{"onlyUpgrade", olds.OnlyUpgrade, news.OnlyUpgrade},
		{"policyRcD", olds.PolicyRcD, news.PolicyRcD},
		{"purge", olds.Purge, news.Purge},
		{"updateCache", olds.UpdateCache, news.UpdateCache},
		{"updateCacheRetries", olds.UpdateCacheRetries, news.UpdateCacheRetries},
		{"updateCacheRetryMaxDelay", olds.UpdateCacheRetryMaxDelay, news.UpdateCacheRetryMaxDelay},
		{"upgrade", olds.Upgrade, news.Upgrade},
	} {
		key := pair[0].(string)
		o := pair[1]
		n := pair[2]

		if n == nil {
			continue
		}

		if o == nil {
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

func (r Apt) Create(
	ctx context.Context,
	name string,
	input AptArgs,
	preview bool,
) (string, AptState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if input.Upgrade == nil && input.UpdateCache == nil && input.Name == nil && input.Names == nil {
		input.Name = ptr.String(name)
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

	if inputs.Name == nil && inputs.Names == nil && state.Name != nil {
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

	if result.IsChanged() && inputs.Upgrade == nil && inputs.UpdateCache == nil {
		if inputs.Ensure != nil && *inputs.Ensure == "absent" {
			// we're going from present? to absent
			if *state.Ensure == "absent" {
				state.Ensure = ptr.String("present")
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

	if news.Upgrade == nil && news.UpdateCache == nil && news.Name == nil && news.Names == nil && olds.Name != nil {
		news.Name = olds.Name
	}

	if news.Upgrade != nil || (news.UpdateCache != nil && news.Name == nil && news.Names == nil) || (news.Ensure != nil && *news.Ensure == "absent") {
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
					"ansible.builtin.apt": parameters,
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
			Name:  absents,
			State: ptr.String("absent"),
		})
	}

	if len(presents) > 0 {
		taskParameterSets = append(taskParameterSets, aptTaskParameters{
			Name:  presents,
			State: ptr.String(newState),
		})
	}

	if len(taskParameterSets) == 0 {
		return olds, errors.New("could not figure out how to update this thing")
	}

	tasks := []any{}
	for _, parameters := range taskParameterSets {
		tasks = append(tasks, map[string]any{
			"ansible.builtin.apt": parameters,
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
	if *props.Ensure == "absent" {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(props.AptArgs)
	parameters.State = ptr.String("absent")
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
