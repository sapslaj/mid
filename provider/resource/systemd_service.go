package resource

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
)

type SystemdService struct{}

type SystemdServiceArgs struct {
	DaemonReexec *bool                `pulumi:"daemonReexec,optional"`
	DaemonReload *bool                `pulumi:"daemonReload,optional"`
	Enabled      *bool                `pulumi:"enabled,optional"`
	Force        *bool                `pulumi:"force,optional"`
	Masked       *bool                `pulumi:"masked,optional"`
	Name         *string              `pulumi:"name,optional"`
	NoBlock      *bool                `pulumi:"noBlock,optional"`
	Scope        *string              `pulumi:"scope,optional"`
	State        *string              `pulumi:"state,optional"` // TODO: enum for this?
	Triggers     *types.TriggersInput `pulumi:"triggers,optional"`
}

type SystemdServiceState struct {
	SystemdServiceArgs
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

type systemdServiceTaskParameters struct {
	DaemonReexec *bool   `json:"daemon_reexec,omitempty"`
	DaemonReload *bool   `json:"daemon_reload,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	Force        *bool   `json:"force,omitempty"`
	Masked       *bool   `json:"masked,omitempty"`
	Name         *string `json:"name,omitempty"`
	NoBlock      *bool   `json:"no_block,omitempty"`
	Scope        *string `json:"scope,omitempty"`
	State        *string `json:"state,omitempty"` // TODO: enum for this?
}

type systemdServiceTaskResult struct {
	Changed *bool   `json:"changed,omitempty"`
	Diff    *any    `json:"diff,omitempty"`
	Msg     *string `json:"msg,omitempty"`
}

func (result *systemdServiceTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r SystemdService) argsToTaskParameters(input SystemdServiceArgs) (systemdServiceTaskParameters, error) {
	return systemdServiceTaskParameters{
		DaemonReexec: input.DaemonReexec,
		DaemonReload: input.DaemonReload,
		Enabled:      input.Enabled,
		Force:        input.Force,
		Masked:       input.Masked,
		Name:         input.Name,
		NoBlock:      input.NoBlock,
		Scope:        input.Scope,
		State:        input.State,
	}, nil
}

func (r SystemdService) updateState(olds SystemdServiceState, news SystemdServiceArgs, changed bool) SystemdServiceState {
	olds.SystemdServiceArgs = news
	if news.Triggers != nil {
		olds.Triggers.Replace = news.Triggers.Replace
		olds.Triggers.Refresh = news.Triggers.Refresh
	}
	if changed {
		olds.Triggers.LastChanged = time.Now().UTC().Format(time.RFC3339)
	}
	return olds
}

func (r SystemdService) Diff(
	ctx context.Context,
	id string,
	olds SystemdServiceState,
	news SystemdServiceArgs,
) (p.DiffResponse, error) {
	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	if news.Name == nil && olds.Name != nil {
		news.Name = olds.Name
	}

	for _, pair := range [][]any{
		{"daemonReexec", olds.DaemonReexec, news.DaemonReexec},
		{"daemonReload", olds.DaemonReload, news.DaemonReload},
		{"enabled", olds.Enabled, news.Enabled},
		{"force", olds.Force, news.Force},
		{"masked", olds.Masked, news.Masked},
		{"name", olds.Name, news.Name},
		{"noBlock", olds.NoBlock, news.NoBlock},
		{"scope", olds.Scope, news.Scope},
		{"state", olds.State, news.State},
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

func (r SystemdService) Create(
	ctx context.Context,
	name string,
	input SystemdServiceArgs,
	preview bool,
) (string, SystemdServiceState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil && (input.Enabled != nil || input.Masked != nil || input.State != nil) {
		input.Name = ptr.String(name)
	}

	state := r.updateState(SystemdServiceState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		return id, state, err
	}

	playOutput, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.systemd_service": parameters,
			},
		},
	})
	if err != nil {
		if preview {
			taskResult, err := executor.GetTaskResult[*systemdServiceTaskResult](playOutput, 0, 0)
			if err == nil && taskResult.Msg != nil && strings.Contains(*taskResult.Msg, "Could not find the requested service") {
				// the service not being available yet might be expected during a preview!
				return id, state, nil
			}
		}
		return id, state, err
	}

	return id, state, nil
}

func (r SystemdService) Read(
	ctx context.Context,
	id string,
	inputs SystemdServiceArgs,
	state SystemdServiceState,
) (string, SystemdServiceArgs, SystemdServiceState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil && state.Name != nil && (inputs.Enabled != nil || inputs.Masked != nil || inputs.State != nil) {
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
				"ansible.builtin.systemd_service": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*systemdServiceTaskResult](output, 0, 0)
	if err != nil {
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	return id, inputs, state, nil
}

func (r SystemdService) Update(
	ctx context.Context,
	id string,
	olds SystemdServiceState,
	news SystemdServiceArgs,
	preview bool,
) (SystemdServiceState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil && olds.Name != nil && (news.Enabled != nil || news.Masked != nil || news.State != nil) {
		news.Name = olds.Name
	}

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		return olds, err
	}

	refreshDiff := resource.NewPropertyValue(olds.Triggers.Refresh).Diff(resource.NewPropertyValue(news.Triggers.Refresh))
	if refreshDiff != nil {
		if news.State != nil && *news.State == "started" {
			parameters.State = ptr.String("restarted")
		}
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.systemd_service": parameters,
			},
		},
	})
	if err != nil {
		return olds, err
	}

	result, err := executor.GetTaskResult[*systemdServiceTaskResult](output, 0, 0)

	state := r.updateState(olds, news, result.IsChanged())
	return state, nil
}

func (r SystemdService) Delete(
	ctx context.Context,
	id string,
	props SystemdServiceState,
) error {
	config := infer.GetConfig[types.Config](ctx)

	args := SystemdServiceArgs{
		DaemonReexec: props.DaemonReexec,
		DaemonReload: props.DaemonReload,
		Enabled:      props.Enabled,
		Force:        props.Force,
		Masked:       props.Masked,
		Name:         props.Name,
		NoBlock:      props.NoBlock,
		Scope:        props.Scope,
		State:        props.State,
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

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.systemd_service": parameters,
			},
		},
	})
	return err
}
