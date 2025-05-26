package resource

import (
	"context"
	"errors"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"
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

func (r FileLine) argsToTaskParameters(input FileLineArgs) (ansible.LineinfileParameters, error) {
	var state *ansible.LineinfileState
	if input.Ensure != nil {
		state = ansible.OptionalLineinfileState(string(*input.Ensure))
	}
	return ansible.LineinfileParameters{
		State:        state,
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
	ctx, span := Tracer.Start(ctx, "mid:resource:FileLine.Diff", trace.WithAttributes(
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

	span.SetStatus(codes.Ok, "")
	return diff, nil
}

func (r FileLine) Create(
	ctx context.Context,
	name string,
	input FileLineArgs,
	preview bool,
) (string, FileLineState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:FileLine.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(FileLineState{}, input, true)

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

	if preview {
		stat, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, config.Connection, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path: input.Path,
			},
		})
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && preview {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return id, state, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return id, state, err
		}
		if !stat.Result.Exists {
			// file doesn't exist yet during preview
			span.SetStatus(codes.Ok, "")
			return id, state, nil
		}
	}

	_, err = executor.AnsibleExecute[
		ansible.LineinfileParameters,
		ansible.LineinfileReturn,
	](ctx, config.Connection, parameters, preview)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && preview {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return id, state, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, state, nil
}

func (r FileLine) Read(
	ctx context.Context,
	id string,
	inputs FileLineArgs,
	state FileLineState,
) (string, FileLineArgs, FileLineState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:FileLine.Read", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	result, err := executor.AnsibleExecute[
		ansible.LineinfileParameters,
		ansible.LineinfileReturn,
	](ctx, config.Connection, parameters, true)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return id, inputs, FileLineState{
				FileLineArgs: inputs,
			}, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r FileLine) Update(
	ctx context.Context,
	id string,
	olds FileLineState,
	news FileLineArgs,
	preview bool,
) (FileLineState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:FileLine.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	if preview {
		stat, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, config.Connection, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path: news.Path,
			},
		})
		if err != nil {
			if errors.Is(err, executor.ErrUnreachable) && preview {
				span.SetAttributes(attribute.Bool("unreachable", true))
				span.SetStatus(codes.Ok, "")
				return olds, nil
			}
			span.SetStatus(codes.Error, err.Error())
			return olds, err
		}
		if !stat.Result.Exists {
			// file doesn't exist yet during preview
			state := r.updateState(olds, news, true)
			span.SetStatus(codes.Ok, "")
			return state, nil
		}
	}

	result, err := executor.AnsibleExecute[
		ansible.LineinfileParameters,
		ansible.LineinfileReturn,
	](ctx, config.Connection, parameters, preview)
	if err != nil {
		if errors.Is(err, executor.ErrUnreachable) && preview {
			span.SetAttributes(attribute.Bool("unreachable", true))
			span.SetStatus(codes.Ok, "")
			return olds, nil
		}
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	state := r.updateState(olds, news, result.IsChanged())

	span.SetStatus(codes.Ok, "")
	return state, nil
}
