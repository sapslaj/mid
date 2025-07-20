package resource

import (
	"context"
	"errors"
	"reflect"
	"slices"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/pdiff"
	"github.com/sapslaj/mid/pkg/providerfw/introspect"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type FileLine struct{}

type FileLineArgs struct {
	Ensure       *string                  `pulumi:"ensure,optional"`
	Path         string                   `pulumi:"path"`
	Backrefs     *bool                    `pulumi:"backrefs,optional"`
	Backup       *bool                    `pulumi:"backup,optional"`
	Create       *bool                    `pulumi:"create,optional"`
	FirstMatch   *bool                    `pulumi:"firstMatch,optional"`
	InsertBefore *string                  `pulumi:"insertBefore,optional"`
	InsertAfter  *string                  `pulumi:"insertAfter,optional"`
	Line         *string                  `pulumi:"line,optional"`
	Regexp       *string                  `pulumi:"regexp,optional"`
	SearchString *string                  `pulumi:"searchString,optional"`
	UnsafeWrites *bool                    `pulumi:"unsafeWrites,optional"`
	Validate     *string                  `pulumi:"validate,optional"`
	Connection   *midtypes.Connection     `pulumi:"connection,optional"`
	Config       *midtypes.ResourceConfig `pulumi:"config,optional"`
	Triggers     *midtypes.TriggersInput  `pulumi:"triggers,optional"`
}

type FileLineState struct {
	FileLineArgs
	Drifted  []string                `pulumi:"_drifted"`
	Triggers midtypes.TriggersOutput `pulumi:"triggers"`
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
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r FileLine) updateStateDrifted(inputs FileLineArgs, state FileLineState, props []string) FileLineState {
	if len(props) > 0 {
		state = r.updateState(inputs, state, true)
	}
	inputsMap := introspect.StructToMap(inputs)
	if state.Drifted == nil {
		state.Drifted = []string{}
	}
	for _, prop := range props {
		val, ok := inputsMap[prop]
		if !ok || val == nil {
			continue
		}
		rv := reflect.ValueOf(val)
		if slices.Contains([]reflect.Kind{
			reflect.Chan,
			reflect.Func,
			reflect.Interface,
			reflect.Map,
			reflect.Pointer,
			reflect.Slice,
		}, rv.Type().Kind()) {
			if rv.IsNil() {
				continue
			}
		}
		if !slices.Contains(state.Drifted, prop) {
			state.Drifted = append(state.Drifted, prop)
		}
	}
	return state
}

func (r FileLine) ansibleLineinfileDiffedAttributes(result ansible.LineinfileReturn) []string {
	if result.Diff == nil {
		return []string{}
	}
	data, ok := (*result.Diff).(map[string]any)
	if !ok {
		return []string{}
	}
	beforeAny, ok := data["before"]
	if !ok {
		return []string{}
	}
	before, ok := beforeAny.(map[string]any)
	if !ok {
		return []string{}
	}
	afterAny, ok := data["after"]
	if !ok {
		return []string{}
	}
	after, ok := afterAny.(map[string]any)
	if !ok {
		return []string{}
	}
	diff := []string{}
	for k := range before {
		if !reflect.DeepEqual(before[k], after[k]) {
			diff = append(diff, k)
		}
	}
	if slices.Contains(diff, "state") {
		diff = append(diff, "ensure")
	}
	if slices.Contains(diff, "content") {
		diff = append(diff, "line")
	}
	return diff
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

	for _, prop := range req.State.Drifted {
		diff.HasChanges = true
		diff.DetailedDiff[prop] = p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: false,
		}
	}

	diff = pdiff.MergeDiffResponses(
		diff,
		pdiff.DiffAllAttributesExcept(req.Inputs, req.State, []string{"triggers"}),
		midtypes.DiffTriggers(req.State, req.Inputs),
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

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

	state := r.updateState(req.Inputs, FileLineState{}, true)
	state.Drifted = []string{}
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
		](ctx, connection, config, rpc.RPCCall[rpc.FileStatArgs]{
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
	](ctx, connection, config, parameters, req.DryRun)
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

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

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
	](ctx, connection, config, parameters, true)
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

	if result.IsChanged() {
		state = r.updateStateDrifted(req.Inputs, state, r.ansibleLineinfileDiffedAttributes(result))
	}

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

	connection := midtypes.GetConnection(ctx, req.Inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, req.Inputs.Config)

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
		](ctx, connection, config, rpc.RPCCall[rpc.FileStatArgs]{
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
	](ctx, connection, config, parameters, req.DryRun)
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

	if result.IsChanged() {
		state = r.updateStateDrifted(req.Inputs, state, r.ansibleLineinfileDiffedAttributes(result))
	} else {
		state = r.updateState(req.Inputs, state, false)
	}

	if !req.DryRun {
		// clear drifted if we aren't doing a dry-run
		state.Drifted = []string{}
	}

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[FileLineState]{
		Output: state,
	}, nil
}
