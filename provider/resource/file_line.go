package resource

import (
	"context"
	"errors"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/telemetry"
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

func (r FileLine) argsToTaskParameters(inputs FileLineArgs) (ansible.LineinfileParameters, error) {
	return ansible.LineinfileParameters{
		State:        ansible.OptionalLineinfileState(inputs.Ensure),
		Path:         inputs.Path,
		Backrefs:     inputs.Backrefs,
		Backup:       inputs.Backup,
		Create:       inputs.Create,
		Firstmatch:   inputs.FirstMatch,
		Insertbefore: inputs.InsertBefore,
		Insertafter:  inputs.InsertAfter,
		Line:         inputs.Line,
		Regexp:       inputs.Regexp,
		SearchString: inputs.SearchString,
		UnsafeWrites: inputs.UnsafeWrites,
		Validate:     inputs.Validate,
	}, nil
}

func (r FileLine) updateState(inputs FileLineArgs, state FileLineState, changed bool) FileLineState {
	state.FileLineArgs = inputs
	state.Triggers = types.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r FileLine) Diff(
	ctx context.Context,
	req infer.DiffRequest[FileLineArgs, FileLineState],
) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/FileLine.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:FileLine"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: true,
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(req.State, req.Inputs, []string{
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
		types.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r FileLine) Create(
	ctx context.Context,
	req infer.CreateRequest[FileLineArgs],
) (infer.CreateResponse[FileLineState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/FileLine.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:FileLine"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(req.Inputs, FileLineState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[FileLineState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[FileLineState]{
			ID:     id,
			Output: state,
		}, err
	}

	if req.DryRun {
		stat, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, config.Connection, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path: req.Inputs.Path,
			},
		})
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.CreateResponse[FileLineState]{
					ID:     id,
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.CreateResponse[FileLineState]{
				ID:     id,
				Output: state,
			}, err
		}
		if !stat.Result.Exists {
			// file doesn't exist yet during req.DryRun
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[FileLineState]{
				ID:     id,
				Output: state,
			}, nil
		}
	}

	_, err = executor.AnsibleExecute[
		ansible.LineinfileParameters,
		ansible.LineinfileReturn,
	](ctx, config.Connection, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.CreateResponse[FileLineState]{
				ID:     id,
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[FileLineState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[FileLineState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r FileLine) Read(
	ctx context.Context,
	req infer.ReadRequest[FileLineArgs, FileLineState],
) (infer.ReadResponse[FileLineArgs, FileLineState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/FileLine.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:FileLine"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[FileLineArgs, FileLineState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	result, err := executor.AnsibleExecute[
		ansible.LineinfileParameters,
		ansible.LineinfileReturn,
	](ctx, config.Connection, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.ReadResponse[FileLineArgs, FileLineState]{
				ID:     req.ID,
				Inputs: req.Inputs,
				State:  state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[FileLineArgs, FileLineState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[FileLineArgs, FileLineState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r FileLine) Update(
	ctx context.Context,
	req infer.UpdateRequest[FileLineArgs, FileLineState],
) (infer.UpdateResponse[FileLineState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/FileLine.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:FileLine"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	parameters, err := r.argsToTaskParameters(req.Inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[FileLineState]{
			Output: state,
		}, err
	}

	if req.DryRun {
		stat, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, config.Connection, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path: req.Inputs.Path,
			},
		})
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return infer.UpdateResponse[FileLineState]{
					Output: state,
				}, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return infer.UpdateResponse[FileLineState]{
				Output: state,
			}, err
		}
		if !stat.Result.Exists {
			// file doesn't exist yet during preview
			state := r.updateState(req.Inputs, state, true)
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[FileLineState]{
				Output: state,
			}, nil
		}
	}

	result, err := executor.AnsibleExecute[
		ansible.LineinfileParameters,
		ansible.LineinfileReturn,
	](ctx, config.Connection, parameters, req.DryRun)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && req.DryRun {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return infer.UpdateResponse[FileLineState]{
				Output: state,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[FileLineState]{
			Output: state,
		}, err
	}

	state = r.updateState(req.Inputs, state, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[FileLineState]{
		Output: state,
	}, nil
}
