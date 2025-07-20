package resource

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	parchive "github.com/pulumi/pulumi/sdk/v3/go/common/resource/archive"
	passet "github.com/pulumi/pulumi/sdk/v3/go/common/resource/asset"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	infertypes "github.com/sapslaj/mid/pkg/providerfw/infer/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/pdiff"
	"github.com/sapslaj/mid/pkg/providerfw/introspect"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/midtypes"
)

type File struct{}

type FileEnsure string

const (
	FileEnsureFile      FileEnsure = "file"
	FileEnsureDirectory FileEnsure = "directory"
	FileEnsureAbsent    FileEnsure = "absent"
	FileEnsureHard      FileEnsure = "hard"
	FileEnsureLink      FileEnsure = "link"
)

type FileArgs struct {
	AccessTime             *string                    `pulumi:"accessTime,optional"`
	AccessTimeFormat       *string                    `pulumi:"accessTimeFormat,optional"`
	Attributes             *string                    `pulumi:"attributes,optional"`
	Backup                 *bool                      `pulumi:"backup,optional"`
	Checksum               *string                    `pulumi:"checksum,optional"`
	Content                *string                    `pulumi:"content,optional"`
	DirectoryMode          *string                    `pulumi:"directoryMode,optional"`
	Ensure                 *FileEnsure                `pulumi:"ensure,optional"`
	Follow                 *bool                      `pulumi:"follow,optional"`
	Force                  *bool                      `pulumi:"force,optional"`
	Group                  *string                    `pulumi:"group,optional"`
	LocalFollow            *bool                      `pulumi:"localFollow,optional"`
	Mode                   *string                    `pulumi:"mode,optional"`
	ModificationTime       *string                    `pulumi:"modificationTime,optional"`
	ModificationTimeFormat *string                    `pulumi:"modificationTimeFormat,optional"`
	Owner                  *string                    `pulumi:"owner,optional"`
	Path                   string                     `pulumi:"path" provider:"replaceOnChanges"`
	Recurse                *bool                      `pulumi:"recurse,optional"`
	RemoteSource           *string                    `pulumi:"remoteSource,optional"`
	Selevel                *string                    `pulumi:"selevel,optional"`
	Serole                 *string                    `pulumi:"serole,optional"`
	Setype                 *string                    `pulumi:"setype,optional"`
	Seuser                 *string                    `pulumi:"seuser,optional"`
	Source                 *infertypes.AssetOrArchive `pulumi:"source,optional"`
	UnsafeWrites           *bool                      `pulumi:"unsafeWrites,optional"`
	Validate               *string                    `pulumi:"validate,optional"`
	Connection             *midtypes.Connection       `pulumi:"connection,optional"`
	Config                 *midtypes.ResourceConfig   `pulumi:"config,optional"`
	Triggers               *midtypes.TriggersInput    `pulumi:"triggers,optional"`
}

type FileState struct {
	FileArgs
	BackupFile *string                 `pulumi:"backupFile,optional"`
	Drifted    []string                `pulumi:"_drifted"`
	Stat       midtypes.FileStatState  `pulumi:"stat"`
	Triggers   midtypes.TriggersOutput `pulumi:"triggers"`
}

func (r File) updateState(inputs FileArgs, state FileState, changed bool) FileState {
	state.FileArgs = inputs
	state.Triggers = midtypes.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r File) updateStateDrifted(inputs FileArgs, state FileState, props []string) FileState {
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

func (r File) inferEnsure(inputs FileArgs, fallback FileEnsure) FileEnsure {
	if inputs.Ensure != nil {
		return *inputs.Ensure
	}
	return fallback
}

func (r File) Check(
	ctx context.Context,
	req infer.CheckRequest,
) (infer.CheckResponse[FileArgs], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Check", trace.WithAttributes(
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.name", req.Name),
		attribute.String("pulumi.old_inputs", fmt.Sprintf("%#v", req.OldInputs)),
		attribute.String("pulumi.new_inputs", fmt.Sprintf("%#v", req.NewInputs)),
	))
	defer span.End()

	inputs, failures, err := infer.DefaultCheck[FileArgs](ctx, req.NewInputs)

	defer span.SetAttributes(
		telemetry.OtelJSON("pulumi.check.inputs", inputs),
		telemetry.OtelJSON("pulumi.check.failures", failures),
	)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CheckResponse[FileArgs]{
			Inputs:   inputs,
			Failures: failures,
		}, err
	}

	if inputs.Content != nil && inputs.Source != nil {
		failures = append(failures, p.CheckFailure{
			Property: "content",
			Reason:   "content and source are mutually exclusive",
		})
	}

	if inputs.Source != nil && inputs.RemoteSource != nil {
		failures = append(failures, p.CheckFailure{
			Property: "source",
			Reason:   "source and remoteSource are mutually exclusive",
		})
	}

	if inputs.Content != nil && inputs.RemoteSource != nil {
		failures = append(failures, p.CheckFailure{
			Property: "content",
			Reason:   "content and remoteSource are mutually exclusive",
		})
	}

	if inputs.Ensure != nil {
		switch *inputs.Ensure {
		case FileEnsureFile:
			break

		case FileEnsureDirectory:
			break

		case FileEnsureAbsent:
			if inputs.Source != nil || inputs.RemoteSource != nil || inputs.Content != nil {
				failures = append(failures, p.CheckFailure{
					Property: "ensure",
					Reason:   "ensure=absent cannot be used with content, source, or remoteSource",
				})
			}

		case FileEnsureHard:
			if inputs.Source != nil || inputs.Content != nil {
				failures = append(failures, p.CheckFailure{
					Property: "ensure",
					Reason:   "ensure=hard can only be used with remoteSource",
				})
			} else if inputs.RemoteSource == nil {
				failures = append(failures, p.CheckFailure{
					Property: "ensure",
					Reason:   "remoteSource must be specified when using ensure=hard",
				})
			}

		case FileEnsureLink:
			if inputs.Source != nil || inputs.Content != nil {
				failures = append(failures, p.CheckFailure{
					Property: "ensure",
					Reason:   "ensure=link can only be used with remoteSource",
				})
			} else if inputs.RemoteSource == nil {
				failures = append(failures, p.CheckFailure{
					Property: "ensure",
					Reason:   "remoteSource must be specified when using ensure=link",
				})
			}

		default:
			failures = append(failures, p.CheckFailure{
				Property: "ensure",
				Reason:   fmt.Sprintf("ensure=%s is invalid; must be one of {present,absent,hard,link}", *inputs.Ensure),
			})

		}
	}

	return infer.CheckResponse[FileArgs]{
		Inputs:   inputs,
		Failures: failures,
	}, nil
}

func (r File) Diff(
	ctx context.Context,
	req infer.DiffRequest[FileArgs, FileState],
) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:Group"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	diff := p.DiffResponse{
		DetailedDiff: map[string]p.PropertyDiff{},
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

func (r File) makeAnsibleCopyParameters(inputs FileArgs, stat rpc.FileStatResult) ansible.CopyParameters {
	parameters := ansible.CopyParameters{
		Attributes:    inputs.Attributes,
		Backup:        inputs.Backup,
		Checksum:      inputs.Checksum,
		Content:       inputs.Content,
		Dest:          inputs.Path,
		DirectoryMode: ptr.ToAny(inputs.DirectoryMode),
		Follow:        inputs.Follow,
		Force:         inputs.Force,
		Group:         inputs.Group,
		LocalFollow:   inputs.LocalFollow,
		Mode:          ptr.ToAny(inputs.Mode),
		Owner:         inputs.Owner,
		RemoteSrc:     ptr.Of(true),
		Selevel:       inputs.Selevel,
		Serole:        inputs.Serole,
		Setype:        inputs.Setype,
		Seuser:        inputs.Seuser,
		Src:           inputs.RemoteSource,
		UnsafeWrites:  inputs.UnsafeWrites,
		Validate:      inputs.Validate,
	}
	// for some insane reason, I need to do this because `copy` will do different
	// things depending on whether the destination is a new file or not. To quote:
	//
	//   - If `mode` is not specified and the destination filesystem object _does
	//     not_ exist, the default `umask` on the system will be used when setting
	//     the mode for the newly created filesystem object.
	//   - If `mode` is not specified and the destination filesystem object
	//     _does_ exist, the mode of the existing filesystem object will be used.
	//
	// Ridiculous.
	if stat.Exists {
		// TODO: support attributes
		if parameters.Mode == nil && stat.FileMode != nil {
			parameters.Mode = ptr.ToAny(ptr.Of("0" + strconv.FormatUint(uint64(*stat.FileMode), 8)))
		}
		if parameters.Group == nil && stat.GroupName != nil {
			parameters.Group = stat.GroupName
		}
		if parameters.Owner == nil && stat.UserName != nil {
			parameters.Owner = stat.UserName
		}
	}
	return parameters
}

func (r File) ansibleFileDiffedAttributes(result ansible.FileReturn) []string {
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
	return diff
}

func (r File) copyNetworkSourceDirectory(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyNetworkSourceDirectory", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	exists := stat.Exists
	forceable := false
	if exists && stat.FileMode != nil && !stat.FileMode.IsDir() {
		forceable = true
	}

	if inputs.Force != nil && *inputs.Force && forceable {
		exists = false
		_, err := executor.AnsibleExecute[
			ansible.FileParameters,
			ansible.FileReturn,
		](
			ctx,
			connection,
			config,
			ansible.FileParameters{
				Follow:       inputs.Follow,
				Force:        inputs.Force,
				Path:         inputs.Path,
				Recurse:      inputs.Recurse,
				State:        ansible.OptionalFileState(ansible.FileStateAbsent),
				UnsafeWrites: inputs.UnsafeWrites,
			},
			false,
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		state = r.updateStateDrifted(inputs, state, []string{"ensure"})
	}

	if !exists {
		mkdirResult, err := executor.AnsibleExecute[
			ansible.FileParameters,
			ansible.FileReturn,
		](
			ctx,
			connection,
			config,
			ansible.FileParameters{
				AccessTime:             inputs.AccessTime,
				AccessTimeFormat:       inputs.AccessTimeFormat,
				Attributes:             inputs.Attributes,
				Follow:                 inputs.Follow,
				Force:                  inputs.Force,
				Group:                  inputs.Group,
				Mode:                   ptr.ToAny(inputs.Mode),
				ModificationTime:       inputs.ModificationTime,
				ModificationTimeFormat: inputs.ModificationTimeFormat,
				Owner:                  inputs.Owner,
				Path:                   inputs.Path,
				Recurse:                inputs.Recurse,
				Selevel:                inputs.Selevel,
				Serole:                 inputs.Serole,
				Setype:                 inputs.Setype,
				Seuser:                 inputs.Seuser,
				State:                  ansible.OptionalFileState(ansible.FileStateDirectory),
				UnsafeWrites:           inputs.UnsafeWrites,
			},
			dryRun,
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if mkdirResult.IsChanged() {
			state = r.updateStateDrifted(inputs, state, r.ansibleFileDiffedAttributes(mkdirResult))
		}
	}

	if !dryRun {
		// TODO: better temp file management
		base := filepath.Base(inputs.Path)
		if base == "." {
			base = ""
		}
		base = "mid-temp-" + base + "-"
		tempfilename, err := resource.NewUniqueHex(base, 8, 63)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		tempfilepath := filepath.Join("/tmp", tempfilename)

		downloadResult, err := executor.AnsibleExecute[
			ansible.GetUrlParameters,
			ansible.GetUrlReturn,
		](
			ctx,
			connection,
			config,
			ansible.GetUrlParameters{
				Checksum:     inputs.Checksum,
				Dest:         tempfilepath,
				Force:        ptr.Of(true),
				Mode:         ptr.ToAny(ptr.Of("0600")),
				UnsafeWrites: inputs.UnsafeWrites,
				Url:          *inputs.RemoteSource,
				// TODO: Ciphers:
				// TODO: ClientCert:
				// TODO: ClientKey:
				// TODO: Decompress:
				// TODO: ForceBasicAuth:
				// TODO: Headers:
				// TODO: HttpAgent:
				// TODO: Timeout:
				// TODO: TmpDest:
				// TODO: UnredirectedHeaders:
				// TODO: UrlPassword:
				// TODO: UrlUsername:
				// TODO: UseGssapi:
				// TODO: UseNetrc:
				// TODO: UseProxy:
				// TODO: ValidateCerts:
			},
			dryRun,
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		if downloadResult.IsChanged() {
			state = r.updateStateDrifted(inputs, state, []string{
				// TODO: filter this down based on resulting diff
				"checksum",
				"remoteSource",
			})
		}

		unarchiveResult, err := executor.AnsibleExecute[
			ansible.UnarchiveParameters,
			ansible.UnarchiveReturn,
		](
			ctx,
			connection,
			config,
			ansible.UnarchiveParameters{
				Attributes:   inputs.Attributes,
				Dest:         inputs.Path,
				Group:        inputs.Group,
				Mode:         ptr.ToAny(inputs.Mode),
				Owner:        inputs.Owner,
				RemoteSrc:    ptr.Of(true),
				Selevel:      inputs.Selevel,
				Serole:       inputs.Serole,
				Setype:       inputs.Setype,
				Seuser:       inputs.Seuser,
				Src:          tempfilepath,
				UnsafeWrites: inputs.UnsafeWrites,
				// TODO: Exclude:
				// TODO: ExtraOpts:
				// TODO: IoBufferSize:
				// TODO: Exclude:
				// TODO: Include:
				// TODO: KeepNewer:
				// TODO: ValidateCerts:
			},
			dryRun,
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		if unarchiveResult.IsChanged() {
			state = r.updateStateDrifted(inputs, state, []string{
				// TODO: filter this down based on resulting diff
				"attributes",
				"group",
				"mode",
				"owner",
				"selevel",
				"serole",
				"setype",
				"seuser",
				"remoteSource",
			})
		}
	} else {
		state = r.updateStateDrifted(inputs, state, []string{
			// it could be any of these, there's no way to know since we aren't in a
			// situation where we can check.
			"attributes",
			"group",
			"mode",
			"owner",
			"selevel",
			"serole",
			"setype",
			"seuser",
			"remoteSource",
		})
	}

	return state, nil
}

func (r File) copyNetworkSourceFile(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyNetworkSourceFile", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	result, err := executor.AnsibleExecute[
		ansible.GetUrlParameters,
		ansible.GetUrlReturn,
	](
		ctx,
		connection,
		config,
		ansible.GetUrlParameters{
			Attributes:   inputs.Attributes,
			Backup:       inputs.Backup,
			Checksum:     inputs.Checksum,
			Dest:         inputs.Path,
			Force:        inputs.Force,
			Group:        inputs.Group,
			Mode:         ptr.ToAny(inputs.Mode),
			Owner:        inputs.Owner,
			Selevel:      inputs.Selevel,
			Serole:       inputs.Serole,
			Setype:       inputs.Setype,
			Seuser:       inputs.Seuser,
			UnsafeWrites: inputs.UnsafeWrites,
			Url:          *inputs.RemoteSource,
			// TODO: Ciphers:
			// TODO: ClientCert:
			// TODO: ClientKey:
			// TODO: Decompress:
			// TODO: ForceBasicAuth:
			// TODO: Headers:
			// TODO: HttpAgent:
			// TODO: Timeout:
			// TODO: TmpDest:
			// TODO: UnredirectedHeaders:
			// TODO: UrlPassword:
			// TODO: UrlUsername:
			// TODO: UseGssapi:
			// TODO: UseNetrc:
			// TODO: UseProxy:
			// TODO: ValidateCerts:
		},
		dryRun,
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if result.IsChanged() {
		state = r.updateStateDrifted(inputs, state, []string{
			// TODO: filter this down based on resulting diff
			"attributes",
			"checksum",
			"group",
			"mode",
			"owner",
			"selevel",
			"serole",
			"setype",
			"seuser",
			"remoteSource",
		})
	}

	return state, nil
}

func (r File) copyRemoteSourceDirectory(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyRemoteSourceDirectory", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	result, err := executor.AnsibleExecute[
		ansible.CopyParameters,
		ansible.CopyReturn,
	](ctx, connection, config, r.makeAnsibleCopyParameters(inputs, stat), dryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if result.IsChanged() {
		state = r.updateStateDrifted(inputs, state, []string{
			// TODO: filter this down based on resulting diff
			"attributes",
			"directoryMode",
			"group",
			"mode",
			"owner",
			"selevel",
			"serole",
			"setype",
			"seuser",
			"remoteSource",
		})
	}

	return state, nil
}

func (r File) copyRemoteSourceFile(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyRemoteSourceFile", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	result, err := executor.AnsibleExecute[
		ansible.CopyParameters,
		ansible.CopyReturn,
	](ctx, connection, config, r.makeAnsibleCopyParameters(inputs, stat), dryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state.BackupFile = result.BackupFile

	if result.IsChanged() {
		state = r.updateStateDrifted(inputs, state, []string{
			// TODO: filter this down based on resulting diff
			"attributes",
			"checksum",
			"directoryMode",
			"group",
			"mode",
			"owner",
			"selevel",
			"serole",
			"setype",
			"seuser",
			"remoteSource",
			"validate",
		})
	}

	return state, nil
}

func (r File) copyRemoteSource(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (ansible.FileState, FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyRemoteSource", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	source := *inputs.RemoteSource
	isNetworkSource := strings.Contains(source, "://")
	span.SetAttributes(attribute.Bool("is_network_source", isNetworkSource))

	var ansibleFileState ansible.FileState
	defer span.SetAttributes(attribute.String("ansible_file_state", string(ansibleFileState)))

	var err error
	if isNetworkSource {
		if r.inferEnsure(inputs, FileEnsureFile) == FileEnsureDirectory {
			ansibleFileState = ansible.FileStateDirectory
			state, err = r.copyNetworkSourceDirectory(ctx, inputs, state, stat, dryRun)
		} else {
			ansibleFileState = ansible.FileStateFile
			state, err = r.copyNetworkSourceFile(ctx, inputs, state, stat, dryRun)
		}
	} else {
		sourceStat, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, connection, config, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path:              source,
				FollowSymlinks:    inputs.Follow != nil && *inputs.Follow,
				CalculateChecksum: false,
			},
		})
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return ansibleFileState, state, err
		}
		if sourceStat.Error != "" {
			err = errors.New(sourceStat.Error)
			span.SetStatus(codes.Error, err.Error())
			return ansibleFileState, state, err
		}

		if !sourceStat.Result.Exists && dryRun {
			state = r.updateState(inputs, state, true)
			span.SetStatus(codes.Ok, "")
			return ansibleFileState, state, nil
		}

		fallbackEnsure := FileEnsureFile
		if sourceStat.Result.FileMode.IsDir() {
			fallbackEnsure = FileEnsureDirectory
		}

		if r.inferEnsure(inputs, fallbackEnsure) == FileEnsureDirectory {
			ansibleFileState = ansible.FileStateDirectory
			state, err = r.copyRemoteSourceDirectory(ctx, inputs, state, stat, dryRun)
		} else {
			ansibleFileState = ansible.FileStateFile
			state, err = r.copyRemoteSourceFile(ctx, inputs, state, stat, dryRun)
		}
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return ansibleFileState, state, err
	}

	span.SetStatus(codes.Ok, "")
	return ansibleFileState, state, nil
}

func (r File) copyLocalSourceArchive(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyLocalSourceArchive", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	archive := inputs.Source.Archive

	span.SetAttributes(
		attribute.String("archive.sig", archive.Sig),
		attribute.String("archive.hash", archive.Hash),
		telemetry.OtelJSON("archive.assets", archive.Assets),
		attribute.String("archive.path", archive.Path),
		attribute.String("archive.uri", archive.URI),
		attribute.Bool("archive.is_assets", archive.IsAssets()),
		attribute.Bool("archive.is_path", archive.IsPath()),
		attribute.Bool("archive.is_uri", archive.IsURI()),
		attribute.Bool("archive.has_contents", archive.HasContents()),
	)

	if dryRun && !archive.HasContents() {
		if archive.Hash == "" {
			state = r.updateStateDrifted(inputs, state, []string{"source"})
			state.Stat.SHA256Checksum = nil
			span.SetStatus(codes.Ok, "")
			return state, nil
		}

		if state.Source != nil && state.Source.Archive != nil && state.Source.Archive.Hash != archive.Hash {
			state = r.updateStateDrifted(inputs, state, []string{"source"})
			span.SetStatus(codes.Ok, "")
			return state, nil
		}
	}

	// TODO: the archive cannot be opened in dry run, so cannot check that the
	// contents match.

	if dryRun && stat.Exists && stat.SHA256Checksum != nil && state.Stat.SHA256Checksum != nil {
		if *state.Stat.SHA256Checksum != *stat.SHA256Checksum {
			state = r.updateStateDrifted(inputs, state, []string{"source"})
		}
		state.Stat = midtypes.FileStatStateFromRPCResult(stat)
	} else if dryRun {
		state = r.updateStateDrifted(inputs, state, []string{"source"})
	} else {
		// FIXME: the TarGZIPArchive format is broken because the gzip.Writer is
		// never closed. This is an upstream Pulumi SDK bug.
		bbuf := bytes.Buffer{}
		zbuf := gzip.NewWriter(&bbuf)

		err := archive.Archive(parchive.TarArchive, zbuf)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		err = zbuf.Close()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		stagedPath, err := executor.StageFile(ctx, connection, config, &bbuf)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		untarResult, err := executor.CallAgent[
			rpc.UntarArgs,
			rpc.UntarResult,
		](ctx, connection, config, rpc.RPCCall[rpc.UntarArgs]{
			RPCFunction: rpc.RPCUntar,
			Args: rpc.UntarArgs{
				SourceFilePath:  stagedPath,
				TargetDirectory: inputs.Path,
			},
		})
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if untarResult.Error != "" {
			err = errors.New(untarResult.Error)
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		state = r.updateStateDrifted(inputs, state, []string{"source"})
	}

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r File) copyLocalSourceAsset(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyLocalSourceAsset", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	var sourceProp string
	var asset *resource.Asset
	var err error
	if inputs.Content != nil {
		sourceProp = "content"
		asset, err = passet.FromText(*inputs.Content)
	} else {
		sourceProp = "source"
		asset = inputs.Source.Asset
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	span.SetAttributes(
		attribute.String("asset.sig", asset.Sig),
		attribute.String("asset.hash", asset.Hash),
		attribute.String("asset.text", asset.Text),
		attribute.String("asset.path", asset.Path),
		attribute.String("asset.uri", asset.URI),
		attribute.Bool("asset.is_text", asset.IsText()),
		attribute.Bool("asset.is_path", asset.IsPath()),
		attribute.Bool("asset.is_uri", asset.IsURI()),
		attribute.Bool("asset.has_contents", asset.HasContents()),
	)

	if dryRun && !asset.HasContents() && asset.Hash == "" {
		state = r.updateStateDrifted(inputs, state, []string{sourceProp})
		state.Stat.SHA256Checksum = nil
		span.SetStatus(codes.Ok, "")
		return state, nil
	}

	if dryRun && stat.Exists && stat.SHA256Checksum != nil {
		if asset.Hash != *stat.SHA256Checksum {
			state = r.updateStateDrifted(inputs, state, []string{sourceProp})
		}

		state.Stat = midtypes.FileStatStateFromRPCResult(stat)
	} else {
		blob, err := asset.Read()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		defer blob.Close()

		stagedPath, err := executor.StageFile(ctx, connection, config, blob)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		parameters := r.makeAnsibleCopyParameters(inputs, stat)
		parameters.Src = &stagedPath

		result, err := executor.AnsibleExecute[
			ansible.CopyParameters,
			ansible.CopyReturn,
		](ctx, connection, config, parameters, dryRun)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		if result.IsChanged() {
			state = r.updateStateDrifted(inputs, state, []string{
				// TODO: filter this down based on resulting diff
				sourceProp,
				"attributes",
				"checksum",
				"directoryMode",
				"group",
				"mode",
				"owner",
				"selevel",
				"serole",
				"setype",
				"seuser",
				"validate",
			})
		}
	}

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r File) copyLocalSource(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	stat rpc.FileStatResult,
	dryRun bool,
) (ansible.FileState, FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.copyLocalSource", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		telemetry.OtelJSON("stat", stat),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	fallbackEnsure := FileEnsureFile
	if inputs.Source != nil && inputs.Source.Archive != nil {
		fallbackEnsure = FileEnsureDirectory
	}

	var ansibleFileState ansible.FileState
	defer span.SetAttributes(attribute.String("ansible_file_state", string(ansibleFileState)))

	ensure := r.inferEnsure(inputs, fallbackEnsure)
	if ensure == FileEnsureDirectory {
		ansibleFileState = ansible.FileStateDirectory
	} else {
		ansibleFileState = ansible.FileStateFile
	}

	var err error
	if ensure == FileEnsureDirectory {
		state, err = r.copyLocalSourceArchive(ctx, inputs, state, stat, dryRun)
	} else {
		state, err = r.copyLocalSourceAsset(ctx, inputs, state, stat, dryRun)
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return ansibleFileState, state, err
	}

	span.SetStatus(codes.Ok, "")
	return ansibleFileState, state, nil
}

func (r File) createOrUpdate(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.createOrUpdate", trace.WithAttributes(
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state.initial", state),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()
	defer span.SetAttributes(telemetry.OtelJSON("state.final", state))

	connection := midtypes.GetConnection(ctx, inputs.Connection)
	config := midtypes.GetResourceConfig(ctx, inputs.Config)

	if executor.PreviewUnreachable(ctx, connection, config, dryRun) {
		span.SetAttributes(attribute.Bool("unreachable", true))
		span.SetStatus(codes.Ok, "")
		state = r.updateState(inputs, state, true)
		return state, nil
	}

	statResult, err := executor.CallAgent[
		rpc.FileStatArgs,
		rpc.FileStatResult,
	](ctx, connection, config, rpc.RPCCall[rpc.FileStatArgs]{
		RPCFunction: rpc.RPCFileStat,
		Args: rpc.FileStatArgs{
			Path:              inputs.Path,
			FollowSymlinks:    inputs.Follow != nil && *inputs.Follow,
			CalculateChecksum: true,
		},
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}
	if statResult.Error != "" {
		err = errors.New(statResult.Error)
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	stat := statResult.Result
	span.SetAttributes(telemetry.OtelJSON("stat.initial", stat))

	var currentState FileEnsure
	if !stat.Exists {
		currentState = FileEnsureAbsent
	} else if stat.FileMode.IsDir() {
		currentState = FileEnsureDirectory
	} else if stat.FileMode.IsRegular() {
		currentState = FileEnsureFile
	} else {
		// TODO: figure out if symlink or hardlink or not
		currentState = FileEnsureAbsent
	}

	if currentState == FileEnsureAbsent && dryRun {
		// exit early if the parent dir doesn't exist
		dir := filepath.Dir(inputs.Path)
		span.SetAttributes(attribute.String("parent_directory.path", dir))

		dirStat, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, connection, config, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path:              dir,
				CalculateChecksum: false,
				FollowSymlinks:    inputs.Follow != nil && *inputs.Follow,
			},
		})
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if dirStat.Error != "" {
			err = errors.New(dirStat.Error)
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		span.SetAttributes(attribute.Bool("parent_directory.exists", dirStat.Result.Exists))

		if !dirStat.Result.Exists {
			span.SetStatus(codes.Ok, "")
			state = r.updateState(inputs, state, true)
			return state, err
		}
	}

	var ansibleDesiredState ansible.FileState
	if inputs.Ensure != nil {
		span.SetAttributes(attribute.Bool("ansible_file_state.inferred", false))
		switch *inputs.Ensure {
		case FileEnsureFile:
			ansibleDesiredState = ansible.FileStateFile
		case FileEnsureDirectory:
			ansibleDesiredState = ansible.FileStateDirectory
		case FileEnsureAbsent:
			ansibleDesiredState = ansible.FileStateAbsent
		case FileEnsureHard:
			ansibleDesiredState = ansible.FileStateHard
		case FileEnsureLink:
			ansibleDesiredState = ansible.FileStateLink
		}
	}

	var fallbackAnsibleState ansible.FileState
	if inputs.RemoteSource != nil {
		fallbackAnsibleState, state, err = r.copyRemoteSource(ctx, inputs, state, stat, dryRun)
	} else if inputs.Source != nil || inputs.Content != nil {
		fallbackAnsibleState, state, err = r.copyLocalSource(ctx, inputs, state, stat, dryRun)
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	if ansibleDesiredState == "" {
		span.SetAttributes(attribute.Bool("ansible_file_state.inferred", true))
		if fallbackAnsibleState != "" {
			ansibleDesiredState = fallbackAnsibleState
		} else {
			ansibleDesiredState = ansible.FileStateTouch
		}
	}

	if fallbackAnsibleState == "" && !stat.Exists && ansibleDesiredState == ansible.FileStateFile {
		// special case to make sure file gets created if it wouldn't otherwise by
		// a copy operation
		ansibleDesiredState = ansible.FileStateTouch
	}

	span.SetAttributes(attribute.String("ansible_file_state", string(ansibleDesiredState)))

	var desiredState FileEnsure
	if inputs.Ensure != nil {
		desiredState = *inputs.Ensure
	} else {
		switch ansibleDesiredState {
		case ansible.FileStateAbsent:
			desiredState = FileEnsureAbsent
		case ansible.FileStateDirectory:
			desiredState = FileEnsureDirectory
		case ansible.FileStateFile:
			desiredState = FileEnsureFile
		case ansible.FileStateHard:
			desiredState = FileEnsureHard
		case ansible.FileStateLink:
			desiredState = FileEnsureLink
		case ansible.FileStateTouch:
			desiredState = FileEnsureFile
		}
	}

	defer span.SetAttributes(
		attribute.String("ensure.current_state", string(currentState)),
		attribute.String("ensure.desired_state", string(desiredState)),
	)

	executeFile := func() error {
		params := ansible.FileParameters{
			AccessTime:             inputs.AccessTime,
			AccessTimeFormat:       inputs.AccessTimeFormat,
			Attributes:             inputs.Attributes,
			Follow:                 inputs.Follow,
			Force:                  inputs.Force,
			Group:                  inputs.Group,
			Mode:                   ptr.ToAny(inputs.Mode),
			ModificationTime:       inputs.ModificationTime,
			ModificationTimeFormat: inputs.ModificationTimeFormat,
			Owner:                  inputs.Owner,
			Path:                   inputs.Path,
			Recurse:                inputs.Recurse,
			Selevel:                inputs.Selevel,
			Serole:                 inputs.Serole,
			Setype:                 inputs.Setype,
			Seuser:                 inputs.Seuser,
			State:                  ansible.OptionalFileState(ansibleDesiredState),
			UnsafeWrites:           inputs.UnsafeWrites,
		}
		if desiredState != FileEnsureDirectory && desiredState != FileEnsureFile {
			params.Src = inputs.RemoteSource
		}
		result, err := executor.AnsibleExecute[
			ansible.FileParameters,
			ansible.FileReturn,
		](ctx, connection, config, params, dryRun)
		if err != nil {
			return err
		}

		if result.IsChanged() {
			state = r.updateStateDrifted(inputs, state, r.ansibleFileDiffedAttributes(result))
		}
		return nil
	}

	if currentState == desiredState {
		err = executeFile()
	} else if desiredState == FileEnsureAbsent {
		err = executeFile()
		state = r.updateStateDrifted(inputs, state, []string{"ensure"})
	} else if currentState == FileEnsureAbsent && !dryRun && ansibleDesiredState != "" {
		err = executeFile()
		state = r.updateStateDrifted(inputs, state, []string{"ensure"})
	} else if dryRun {
		state = r.updateStateDrifted(inputs, state, []string{
			// it could be any of these, there's no way to know since we aren't in a
			// situation where we can check.
			"accessTime",
			"attributes",
			"group",
			"mode",
			"modificationTime",
			"owner",
			"selevel",
			"serole",
			"setype",
			"seuser",
			"remoteSource",
		})
	} else if inputs.Force != nil && *inputs.Force {
		if ansibleDesiredState == ansible.FileStateFile {
			ansibleDesiredState = ansible.FileStateTouch
		}
		_, err = executor.AnsibleExecute[
			ansible.FileParameters,
			ansible.FileReturn,
		](
			ctx,
			connection,
			config,
			ansible.FileParameters{
				Follow:       inputs.Follow,
				Force:        inputs.Force,
				Path:         inputs.Path,
				Recurse:      inputs.Recurse,
				State:        ansible.OptionalFileState(ansible.FileStateAbsent),
				UnsafeWrites: inputs.UnsafeWrites,
			},
			false,
		)
		err = errors.Join(err, executeFile())
		state = r.updateStateDrifted(inputs, state, []string{"ensure"})
	} else {
		err = fmt.Errorf(
			"unsupported situation (this is a bug): current_state=%v desired_state=%v",
			currentState,
			desiredState,
		)
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	span.SetAttributes(attribute.StringSlice("drifted", state.Drifted))

	if !dryRun {
		// clear drifted if we aren't doing a dry-run
		state.Drifted = []string{}

		statResult, err := executor.CallAgent[
			rpc.FileStatArgs,
			rpc.FileStatResult,
		](ctx, connection, config, rpc.RPCCall[rpc.FileStatArgs]{
			RPCFunction: rpc.RPCFileStat,
			Args: rpc.FileStatArgs{
				Path:              inputs.Path,
				FollowSymlinks:    inputs.Follow != nil && *inputs.Follow,
				CalculateChecksum: true,
			},
		})
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if statResult.Error != "" {
			err = errors.New(statResult.Error)
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		state.Stat = midtypes.FileStatStateFromRPCResult(statResult.Result)
		span.SetAttributes(telemetry.OtelJSON("stat.final", stat))
	}

	return state, nil
}

func (r File) Create(
	ctx context.Context,
	req infer.CreateRequest[FileArgs],
) (infer.CreateResponse[FileState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Create", trace.WithAttributes(
		attribute.String("pulumi.operation", "create"),
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.name", req.Name),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	state := r.updateState(req.Inputs, FileState{}, true)
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	id, err := resource.NewUniqueHex(req.Name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[FileState]{
			ID:     id,
			Output: state,
		}, err
	}
	span.SetAttributes(attribute.String("pulumi.id", id))

	state, err = r.createOrUpdate(ctx, req.Inputs, state, req.DryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.CreateResponse[FileState]{
			ID:     id,
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.CreateResponse[FileState]{
		ID:     id,
		Output: state,
	}, nil
}

func (r File) Read(
	ctx context.Context,
	req infer.ReadRequest[FileArgs, FileState],
) (infer.ReadResponse[FileArgs, FileState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	state, err := r.createOrUpdate(ctx, req.Inputs, state, true)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.ReadResponse[FileArgs, FileState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.ReadResponse[FileArgs, FileState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (r File) Update(
	ctx context.Context,
	req infer.UpdateRequest[FileArgs, FileState],
) (infer.UpdateResponse[FileState], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Update", trace.WithAttributes(
		attribute.String("pulumi.operation", "update"),
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
		attribute.Bool("pulumi.dry_run", req.DryRun),
	))
	defer span.End()

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	state, err := r.createOrUpdate(ctx, req.Inputs, state, req.DryRun)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.UpdateResponse[FileState]{
			Output: state,
		}, err
	}

	span.SetStatus(codes.Ok, "")
	return infer.UpdateResponse[FileState]{
		Output: state,
	}, nil
}

func (r File) Delete(
	ctx context.Context,
	req infer.DeleteRequest[FileState],
) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	connection := midtypes.GetConnection(ctx, req.State.Connection)
	config := midtypes.GetResourceConfig(ctx, req.State.Config)

	_, err := executor.AnsibleExecute[
		ansible.FileParameters,
		ansible.FileReturn,
	](
		ctx,
		connection,
		config,
		ansible.FileParameters{
			Path:  req.State.Path,
			State: ansible.OptionalFileState(ansible.FileStateAbsent),
		},
		false,
	)
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
