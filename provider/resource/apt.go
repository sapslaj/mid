package resource

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
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

func (r Apt) taskParametersNeedsName(input AptArgs) bool {
	return !ptr.AnyNonNils(
		input.Autoclean,
		input.Autoremove,
		input.Clean,
		input.Deb,
		input.UpdateCache,
		input.Upgrade,
	)
}

func (r Apt) canAssumeEnsure(input AptArgs) bool {
	if ptr.AnyNonNils(
		input.Name,
		input.Names,
		input.Deb,
	) {
		return true
	}

	return r.taskParametersNeedsName(input)
}

func (r Apt) argsToTaskParameters(input AptArgs) (ansible.AptParameters, error) {
	parameters := ansible.AptParameters{
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
		State:                    ansible.OptionalAptState(input.Ensure),
		UpdateCache:              input.UpdateCache,
		UpdateCacheRetries:       input.UpdateCacheRetries,
		UpdateCacheRetryMaxDelay: input.UpdateCacheRetryMaxDelay,
		Upgrade:                  ansible.OptionalAptUpgrade(input.Upgrade),
	}

	if input.Name != nil && parameters.Name == nil {
		parameters.Name = ptr.Of([]string{*input.Name})
	}

	if parameters.LockTimeout == nil {
		parameters.LockTimeout = ptr.Of(120)
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

func (r Apt) runApt(
	ctx context.Context,
	connection *types.Connection,
	parameters ansible.AptParameters,
	preview bool,
) (ansible.AptReturn, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Apt.runApt", trace.WithAttributes(
		attribute.String("connection.host", *connection.Host),
		telemetry.OtelJSON("parameters", parameters),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	var err error
	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return ansible.AptReturn{}, err
	}
	call.Args.Check = preview

	var callResult rpc.RPCResult[rpc.AnsibleExecuteResult]
	var result ansible.AptReturn
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt == 10 {
			break
		}

		attemptCtx, attemptSpan := Tracer.Start(ctx, "mid:resource:Apt.runApt:Attempt", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
		))

		// TODO: maybe refactor this to use executor.AnsibleExecute?
		callResult, err = executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](attemptCtx, connection, call)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && preview {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return ansible.AptReturn{
					AnsibleCommonReturns: ansible.AnsibleCommonReturns{
						Changed: true,
						Failed:  false,
						Msg:     ptr.Of(err.Error()),
					},
				}, nil
			}
			attemptSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, err.Error())
			attemptSpan.End()
			return ansible.AptReturn{}, err
		}

		result, err = ansible.AptReturnFromRPCResult(callResult)
		if err != nil {
			attemptSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, err.Error())
			attemptSpan.End()
			return ansible.AptReturn{}, err
		}

		shouldRetry := false

		if callResult.Result.Success {
			attemptSpan.SetStatus(codes.Ok, "")
			span.SetStatus(codes.Ok, "")
			attemptSpan.End()
			break
		}

		errorStr := "running apt failed:"
		errorStr += fmt.Sprintf(" call_exitcode=%d", callResult.Result.ExitCode)
		errorStr += fmt.Sprintf(" call_stderr=%s", string(callResult.Result.Stderr))
		errorStr += fmt.Sprintf(" call_stdout=%s", string(callResult.Result.Stdout))
		if result.Stderr != nil {
			errorStr += fmt.Sprintf(" apt_stderr=%s", string(*result.Stderr))
		}
		if result.Stdout != nil {
			errorStr += fmt.Sprintf(" apt_stdout=%s", string(*result.Stdout))
		}
		attemptSpan.SetStatus(codes.Error, errorStr)
		span.SetStatus(codes.Error, errorStr)
		err = errors.New(errorStr)

		attemptSpan.End()

		if result.Stderr != nil && strings.Contains(string(*result.Stderr), "Unable to acquire the dpkg frontend lock") {
			shouldRetry = true
		}

		if !shouldRetry {
			break
		}

		time.Sleep(time.Duration(attempt) * 10 * time.Second)
	}

	return result, err
}

func (r Apt) Diff(
	ctx context.Context,
	id string,
	olds AptState,
	news AptArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Apt.Diff", trace.WithAttributes(
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
		news.Ensure = ptr.Of("present")
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

	span.SetStatus(codes.Ok, "")
	return diff, nil
}

func (r Apt) Create(
	ctx context.Context,
	name string,
	input AptArgs,
	preview bool,
) (string, AptState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Apt.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if r.taskParametersNeedsName(input) && input.Name == nil && input.Names == nil {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(AptState{}, input, true)

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

	_, err = r.runApt(ctx, config.Connection, parameters, preview)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, state, nil
}

func (r Apt) Read(
	ctx context.Context,
	id string,
	inputs AptArgs,
	state AptState,
) (string, AptArgs, AptState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Apt.Read", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if r.taskParametersNeedsName(inputs) && inputs.Name == nil && inputs.Names == nil && state.Name != nil {
		inputs.Name = state.Name
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	result, err := r.runApt(ctx, config.Connection, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return id, inputs, AptState{
				AptArgs: inputs,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
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

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r Apt) Update(
	ctx context.Context,
	id string,
	olds AptState,
	news AptArgs,
	preview bool,
) (AptState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:Apt.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	if r.taskParametersNeedsName(news) && news.Name == nil && news.Names == nil && olds.Name != nil {
		news.Name = olds.Name
	}

	if (news.Ensure != nil && *news.Ensure == "absent") || !r.canAssumeEnsure(news) {
		parameters, err := r.argsToTaskParameters(news)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return olds, err
		}

		result, err := r.runApt(ctx, config.Connection, parameters, preview)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return olds, err
		}

		state := r.updateState(olds, news, result.IsChanged())
		span.SetStatus(codes.Ok, "")
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
		err := errors.New("we somehow forgot the apt name, oops")
		span.SetStatus(codes.Error, err.Error())
		return AptState{}, err
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

	absents := []string{}
	presents := []string{}

	for name, state := range aptStateMap {
		if state == "absent" {
			absents = append(absents, name)
		} else {
			presents = append(presents, name)
		}
	}

	changed := false

	if len(absents) > 0 {
		parameters, err := r.argsToTaskParameters(news)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return AptState{}, err
		}
		parameters.Name = &absents
		parameters.State = ansible.OptionalAptState("absent")
		result, err := r.runApt(ctx, config.Connection, parameters, preview)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return AptState{}, err
		}
		if result.IsChanged() {
			changed = true
		}
	}

	if len(presents) > 0 {
		parameters, err := r.argsToTaskParameters(news)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return AptState{}, err
		}
		parameters.Name = &presents
		parameters.State = ansible.OptionalAptState(newState)
		result, err := r.runApt(ctx, config.Connection, parameters, preview)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return AptState{}, err
		}
		if result.IsChanged() {
			changed = true
		}
	}

	state := r.updateState(olds, news, changed)
	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r Apt) Delete(ctx context.Context, id string, props AptState) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:Apt.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	if !r.taskParametersNeedsName(props.AptArgs) {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	if props.Ensure != nil && *props.Ensure == "absent" {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(props.AptArgs)
	parameters.State = ansible.OptionalAptState("absent")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = r.runApt(ctx, config.Connection, parameters, false)
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
