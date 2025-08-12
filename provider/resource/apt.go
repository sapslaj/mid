package resource

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/pdiff"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type Apt struct{}

type AptArgs struct {
	Name                     *string                  `pulumi:"name,optional"`
	Names                    *[]string                `pulumi:"names,optional"`
	Ensure                   *string                  `pulumi:"ensure,optional"`
	AllowChangeHeldPackages  *bool                    `pulumi:"allowChangeHeldPackages,optional"`
	AllowDowngrade           *bool                    `pulumi:"allowDowngrade,optional"`
	AllowUnauthenticated     *bool                    `pulumi:"allowUnauthenticated,optional"`
	Autoclean                *bool                    `pulumi:"autoclean,optional"`
	Autoremove               *bool                    `pulumi:"autoremove,optional"`
	CacheValidTime           *int                     `pulumi:"cacheValidTime,optional"`
	Clean                    *bool                    `pulumi:"clean,optional"`
	Deb                      *string                  `pulumi:"deb,optional"`
	DefaultRelease           *string                  `pulumi:"defaultRelease,optional"`
	DpkgOptions              *string                  `pulumi:"dpkgOptions,optional"`
	FailOnAutoremove         *bool                    `pulumi:"failOnAutoremove,optional"`
	Force                    *bool                    `pulumi:"force,optional"`
	ForceAptGet              *bool                    `pulumi:"forceAptGet,optional"`
	InstallRecommends        *bool                    `pulumi:"installRecommends,optional"`
	LockTimeout              *int                     `pulumi:"lockTimeout,optional"`
	OnlyUpgrade              *bool                    `pulumi:"onlyUpgrade,optional"`
	PolicyRcD                *int                     `pulumi:"policyRcD,optional"`
	Purge                    *bool                    `pulumi:"purge,optional"`
	UpdateCache              *bool                    `pulumi:"updateCache,optional"`
	UpdateCacheRetries       *int                     `pulumi:"updateCacheRetries,optional"`
	UpdateCacheRetryMaxDelay *int                     `pulumi:"updateCacheRetryMaxDelay,optional"`
	Upgrade                  *string                  `pulumi:"upgrade,optional"`
	Connection               *midtypes.Connection     `pulumi:"connection,optional"`
	Config                   *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers                 *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type AptState struct {
	AptArgs
	PackagesTracked []string                `pulumi:"packagesTracked"`
	Triggers        midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r Apt) canAssumeEnsure(inputs AptArgs) bool {
	if ptr.AnyNonNils(
		inputs.Name,
		inputs.Names,
		inputs.Deb,
	) {
		return true
	}

	return !ptr.AnyNonNils(
		inputs.Autoclean,
		inputs.Autoremove,
		inputs.Clean,
		inputs.Deb,
		inputs.UpdateCache,
		inputs.Upgrade,
	)
}

func (r Apt) argsToTaskParameters(inputs AptArgs) (ansible.AptParameters, error) {
	parameters := ansible.AptParameters{
		AllowChangeHeldPackages:  inputs.AllowChangeHeldPackages,
		AllowDowngrade:           inputs.AllowDowngrade,
		AllowUnauthenticated:     inputs.AllowUnauthenticated,
		Autoclean:                inputs.Autoclean,
		Autoremove:               inputs.Autoremove,
		CacheValidTime:           inputs.CacheValidTime,
		Clean:                    inputs.Clean,
		Deb:                      inputs.Deb,
		DefaultRelease:           inputs.DefaultRelease,
		DpkgOptions:              inputs.DpkgOptions,
		FailOnAutoremove:         inputs.FailOnAutoremove,
		Force:                    inputs.Force,
		ForceAptGet:              inputs.ForceAptGet,
		InstallRecommends:        inputs.InstallRecommends,
		LockTimeout:              inputs.LockTimeout,
		Name:                     inputs.Names,
		OnlyUpgrade:              inputs.OnlyUpgrade,
		PolicyRcD:                inputs.PolicyRcD,
		Purge:                    inputs.Purge,
		State:                    ansible.OptionalAptState(inputs.Ensure),
		UpdateCache:              inputs.UpdateCache,
		UpdateCacheRetries:       inputs.UpdateCacheRetries,
		UpdateCacheRetryMaxDelay: inputs.UpdateCacheRetryMaxDelay,
		Upgrade:                  ansible.OptionalAptUpgrade(inputs.Upgrade),
	}

	if inputs.Name != nil && parameters.Name == nil {
		parameters.Name = ptr.Of([]string{*inputs.Name})
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
	olds.Triggers = midtypes.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Apt) runApt(
	ctx context.Context,
	connection midtypes.Connection,
	config midtypes.ResourceConfig,
	parameters ansible.AptParameters,
	dryRun bool,
) (ansible.AptReturn, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Apt.runApt", trace.WithAttributes(
		telemetry.OtelJSON("parameters", parameters),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()

	if connection.Host != nil {
		span.SetAttributes(
			attribute.String("connection.host", *connection.Host),
		)
	}

	returnFailed := func(err error) ansible.AptReturn {
		return ansible.AptReturn{
			AnsibleCommonReturns: ansible.AnsibleCommonReturns{
				Changed: true,
				Failed:  true,
				Msg:     ptr.Of(err.Error()),
			},
		}
	}
	returnUnreachable := func(err error) ansible.AptReturn {
		if err == nil {
			err = executor.ErrUnreachable
		}
		return ansible.AptReturn{
			AnsibleCommonReturns: ansible.AnsibleCommonReturns{
				Changed: true,
				Failed:  false,
				Msg:     ptr.Of(err.Error()),
			},
		}
	}

	var err error
	call, err := parameters.ToRPCCall()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return returnFailed(err), err
	}
	call.Args.Check = dryRun

	if executor.PreviewUnreachable(ctx, connection, config, dryRun) {
		span.SetAttributes(attribute.Bool("unreachable", true))
		span.SetStatus(codes.Ok, "")
		return returnUnreachable(err), nil
	}

	var callResult rpc.RPCResult[rpc.AnsibleExecuteResult]
	var result ansible.AptReturn
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt == 10 {
			break
		}

		attemptCtx, attemptSpan := Tracer.Start(ctx, "mid/provider/resource/Apt.runApt:Attempt", trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
		))

		// TODO: maybe refactor this to use executor.AnsibleExecute?
		callResult, err = executor.CallAgent[
			rpc.AnsibleExecuteArgs,
			rpc.AnsibleExecuteResult,
		](attemptCtx, connection, config, call)
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && dryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return returnUnreachable(err), nil
			}
			attemptSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, err.Error())
			attemptSpan.End()
			return returnFailed(err), err
		}

		result, err = ansible.AptReturnFromRPCResult(callResult)
		if err != nil {
			attemptSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, err.Error())
			attemptSpan.End()
			return returnFailed(err), err
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
		if strings.Contains(result.GetMsg(), "apt-get clean failed") {
			shouldRetry = true
		}
		if strings.Contains(result.GetMsg(), "Problem renaming the file") {
			shouldRetry = true
		}
		if strings.Contains(result.GetMsg(), "Could not get lock") {
			shouldRetry = true
		}

		if !shouldRetry {
			break
		}

		time.Sleep(time.Duration(attempt) * 10 * time.Second)
	}

	return result, err
}

func (r Apt) Diff(ctx context.Context, req infer.DiffRequest[AptArgs, AptState]) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Apt.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:Apt"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	diff := p.DiffResponse{
		DetailedDiff: map[string]p.PropertyDiff{},
	}

	diff = pdiff.MergeDiffResponses(
		diff,
		pdiff.DiffAllAttributesExcept(req.Inputs, req.State, []string{
			"connection",
			"config",
			"triggers",
		}),
		midtypes.DiffTriggers(req.State, req.Inputs),
	)

	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	span.SetStatus(codes.Ok, "")
	return diff, nil
}

func (r Apt) Create(ctx context.Context, req infer.CreateRequest[AptArgs]) (infer.CreateResponse[AptState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Apt.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:Apt"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(AptState{
		PackagesTracked: []string{},
	}, req.Inputs, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[AptState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	if req.DryRun && !config.GetDryRunCheck() {
		span.SetStatus(codes.Ok, "")
		return infer.CreateResponse[AptState]{
			ID:     id,
			Output: state,
		}, nil
	}

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[AptState]{
			ID:     id,
			Output: state,
		}, err
	}

	result, err := r.runApt(ctx, connection, config, parameters, req.DryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[AptState]{
			ID:     id,
			Output: state,
		}, err
	}

	if result.PackagesTracked != nil {
		state.PackagesTracked = *result.PackagesTracked
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[AptState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r Apt) Read(
	ctx context.Context,
	req infer.ReadRequest[AptArgs, AptState],
) (infer.ReadResponse[AptArgs, AptState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Apt.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:Apt"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[AptArgs, AptState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := r.runApt(ctx, connection, config, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[AptArgs, AptState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, err
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[AptArgs, AptState]{}, err
	}

	if result.PackagesTracked != nil {
		state.PackagesTracked = *result.PackagesTracked
	}

	state = r.updateState(state, req.Inputs, result.IsChanged())

	if result.IsChanged() && r.canAssumeEnsure(req.Inputs) {
		if req.Inputs.Ensure != nil && *req.Inputs.Ensure == "absent" {
			if state.Ensure == nil || *state.Ensure == "absent" {
				state.Ensure = ptr.Of("present")
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[AptArgs, AptState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r Apt) Update(
	ctx context.Context,
	req infer.UpdateRequest[AptArgs, AptState],
) (infer.UpdateResponse[AptState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Apt.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:Apt"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	if req.DryRun && !config.GetDryRunCheck() {
		state = r.updateState(state, req.Inputs, true)
		span.SetStatus(codes.Ok, "")
		return infer.UpdateResponse[AptState]{
			Output: state,
		}, nil
	}

	if (req.Inputs.Ensure != nil && *req.Inputs.Ensure == "absent") || !r.canAssumeEnsure(req.Inputs) {
		parameters, err := r.argsToTaskParameters(req.Inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}

		result, err := r.runApt(ctx, connection, config, parameters, req.DryRun)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}

		if result.PackagesTracked != nil {
			state.PackagesTracked = *result.PackagesTracked
		}

		state := r.updateState(state, req.Inputs, result.IsChanged())
		span.SetStatus(codes.Ok, "")
		return infer.UpdateResponse[AptState]{
			Output: state,
		}, nil
	}

	if req.Inputs.Deb != nil {
		// FIXME: need to detect if we switch from specifying `name`/`names` to
		// `deb` and vice versa.

		parameters, err := r.argsToTaskParameters(req.Inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}

		result, err := r.runApt(ctx, connection, config, parameters, req.DryRun)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}

		if result.PackagesTracked != nil {
			state.PackagesTracked = *result.PackagesTracked
		}

		state := r.updateState(state, req.Inputs, result.IsChanged())
		span.SetStatus(codes.Ok, "")
		return infer.UpdateResponse[AptState]{
			Output: state,
		}, nil
	}

	aptStateMap := map[string]string{}

	newState := "present"
	if state.Ensure != nil {
		newState = *state.Ensure
	}
	if req.Inputs.Ensure != nil {
		newState = *req.Inputs.Ensure
	}

	if req.Inputs.Name != nil {
		aptStateMap[*req.Inputs.Name] = newState
	} else if req.Inputs.Names != nil {
		for _, name := range *req.Inputs.Names {
			aptStateMap[name] = newState
		}
	} else if state.Name != nil {
		aptStateMap[*state.Name] = newState
	} else if state.Names != nil {
		for _, name := range *state.Names {
			aptStateMap[name] = newState
		}
	} else {
		err := errors.New("we somehow forgot the apt name, oops")
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[AptState]{
			Output: state,
		}, err
	}

	if state.Name != nil {
		if _, exists := aptStateMap[*state.Name]; !exists {
			aptStateMap[*state.Name] = "absent"
		}
	} else {
		for _, name := range *state.Names {
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
		parameters, err := r.argsToTaskParameters(req.Inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}
		parameters.Name = &absents
		parameters.State = ansible.OptionalAptState("absent")
		result, err := r.runApt(ctx, connection, config, parameters, req.DryRun)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}
		if result.IsChanged() {
			changed = true
		}
		if result.PackagesTracked != nil {
			state.PackagesTracked = slices.DeleteFunc(state.PackagesTracked, func(tracked string) bool {
				return slices.Contains(*result.PackagesTracked, tracked)
			})
		}
	}

	if len(presents) > 0 {
		parameters, err := r.argsToTaskParameters(req.Inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}
		parameters.Name = &presents
		parameters.State = ansible.OptionalAptState(newState)
		result, err := r.runApt(ctx, connection, config, parameters, req.DryRun)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[AptState]{
				Output: state,
			}, err
		}
		if result.IsChanged() {
			changed = true
		}
		if result.PackagesTracked != nil {
			for _, tracked := range *result.PackagesTracked {
				if !slices.Contains(state.PackagesTracked, tracked) {
					state.PackagesTracked = append(state.PackagesTracked, tracked)
				}
			}
		}
	}

	state = r.updateState(state, req.Inputs, changed)
	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[AptState]{
		Output: state,
	}, nil
}

func (r Apt) Delete(ctx context.Context, req infer.DeleteRequest[AptState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/Apt.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:Apt"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	if !r.canAssumeEnsure(req.State.AptArgs) {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	if req.State.Ensure != nil && *req.State.Ensure == "absent" {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	connection := midtypes.GetConnection(ctx, req.State.Connection)
	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	parameters, err := r.argsToTaskParameters(req.State.AptArgs)
	parameters.State = ansible.OptionalAptState("absent")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}

	if parameters.Deb != nil {
		parameters.Deb = nil
		parameters.Name = ptr.Of(req.State.PackagesTracked)
	}

	_, err = r.runApt(ctx, connection, config, parameters, false)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && config.GetDeleteUnreachable() {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetAttributes(attribute.Bool("unreachable.deleted", true))
			span.SetStatus(codes.Ok, "")
			return infer.DeleteResponse{}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.DeleteResponse{}, nil
}
