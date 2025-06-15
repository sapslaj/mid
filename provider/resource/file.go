package resource

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	ptypes "github.com/pulumi/pulumi-go-provider/infer/types"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	parchive "github.com/pulumi/pulumi/sdk/v3/go/common/resource/archive"
	passet "github.com/pulumi/pulumi/sdk/v3/go/common/resource/asset"
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

type File struct{}

type FileEnsure string

const (
	FileEnsureAbsent    FileEnsure = "absent"
	FileEnsureDirectory FileEnsure = "directory"
	FileEnsureFile      FileEnsure = "file"
	FileEnsureHard      FileEnsure = "hard"
	FileEnsureLink      FileEnsure = "link"
	FileEnsureTouch     FileEnsure = "touch"
)

type FileArgs struct {
	AccessTime             *string                `pulumi:"accessTime,optional"`
	AccessTimeFormat       *string                `pulumi:"accessTimeFormat,optional"`
	Attributes             *string                `pulumi:"attributes,optional"`
	Backup                 *bool                  `pulumi:"backup,optional"`
	Checksum               *string                `pulumi:"checksum,optional"`
	Content                *string                `pulumi:"content,optional"`
	DirectoryMode          *string                `pulumi:"directoryMode,optional"`
	Ensure                 *FileEnsure            `pulumi:"ensure,optional"`
	Follow                 *bool                  `pulumi:"follow,optional"`
	Force                  *bool                  `pulumi:"force,optional"`
	Group                  *string                `pulumi:"group,optional"`
	LocalFollow            *bool                  `pulumi:"localFollow,optional"`
	Mode                   *string                `pulumi:"mode,optional"`
	ModificationTime       *string                `pulumi:"modificationTime,optional"`
	ModificationTimeFormat *string                `pulumi:"modificationTimeFormat,optional"`
	Owner                  *string                `pulumi:"owner,optional"`
	Path                   string                 `pulumi:"path"`
	Recurse                *bool                  `pulumi:"recurse,optional"`
	RemoteSource           *string                `pulumi:"remoteSource,optional"`
	Selevel                *string                `pulumi:"selevel,optional"`
	Serole                 *string                `pulumi:"serole,optional"`
	Setype                 *string                `pulumi:"setype,optional"`
	Seuser                 *string                `pulumi:"seuser,optional"`
	Source                 *ptypes.AssetOrArchive `pulumi:"source,optional"`
	UnsafeWrites           *bool                  `pulumi:"unsafeWrites,optional"`
	Validate               *string                `pulumi:"validate,optional"`
	Triggers               *types.TriggersInput   `pulumi:"triggers,optional"`
}

type FileStateStat struct {
	Atime      float64  `pulumi:"atime" json:"atime"`
	Attributes []string `pulumi:"attributes" json:"attributes"`
	Charset    string   `pulumi:"charset" json:"charset"`
	Checksum   string   `pulumi:"checksum" json:"checksum"`
	Ctime      float64  `pulumi:"ctime" json:"ctime"`
	Dev        int      `pulumi:"dev" json:"dev"`
	Executable bool     `pulumi:"executable" json:"executable"`
	Exists     bool     `pulumi:"exists" json:"exists"`
	Gid        int      `pulumi:"gid" json:"gid"`
	GrName     string   `pulumi:"gr_name" json:"gr_name"`
	Inode      int      `pulumi:"inode" json:"inode"`
	Isblk      bool     `pulumi:"isblk" json:"isblk"`
	Ischr      bool     `pulumi:"ischr" json:"ischr"`
	Isdir      bool     `pulumi:"isdir" json:"isdir"`
	Isfifo     bool     `pulumi:"isfifo" json:"isfifo"`
	Isgid      bool     `pulumi:"isgid" json:"isgid"`
	Islnk      bool     `pulumi:"islnk" json:"islnk"`
	Isreg      bool     `pulumi:"isreg" json:"isreg"`
	Issock     bool     `pulumi:"issock" json:"issock"`
	Isuid      bool     `pulumi:"isuid" json:"isuid"`
	LnkSource  string   `pulumi:"lnkSource" json:"lnk_source"`
	LnkTarget  string   `pulumi:"lnkTarget" json:"lnk_target"`
	Mimetype   string   `pulumi:"mimetype" json:"mimetype"`
	Mode       string   `pulumi:"mode" json:"mode"`
	Mtime      float64  `pulumi:"mtime" json:"mtime"`
	Nlink      int      `pulumi:"nlink" json:"nlink"`
	Path       string   `pulumi:"path" json:"path"`
	PwName     string   `pulumi:"pwName" json:"pw_name"`
	Readable   bool     `pulumi:"readable" json:"readable"`
	Rgrp       bool     `pulumi:"rgrp" json:"rgrp"`
	Roth       bool     `pulumi:"roth" json:"roth"`
	Rusr       bool     `pulumi:"rusr" json:"rusr"`
	Size       int      `pulumi:"size" json:"size"`
	Uid        int      `pulumi:"uid" json:"uid"`
	Version    string   `pulumi:"version" json:"version"`
	Wgrp       bool     `pulumi:"wgrp" json:"wgrp"`
	Woth       bool     `pulumi:"woth" json:"woth"`
	Writeable  bool     `pulumi:"writeable" json:"writeable"`
	Wusr       bool     `pulumi:"wusr" json:"wusr"`
	Xgrp       bool     `pulumi:"xgrp" json:"xgrp"`
	Xoth       bool     `pulumi:"xoth" json:"xoth"`
	Xusr       bool     `pulumi:"xusr" json:"xusr"`
}

type FileState struct {
	FileArgs
	BackupFile *string              `pulumi:"backupFile,optional"`
	Stat       FileStateStat        `pulumi:"stat"`
	Triggers   types.TriggersOutput `pulumi:"triggers"`
}

type FileCopyPlanStrategy string

const (
	FileCopyPlanInvalid         FileCopyPlanStrategy = ""
	FileCopyPlanNop             FileCopyPlanStrategy = "Nop"
	FileCopyPlanChecksumCompare FileCopyPlanStrategy = "ChecksumCompare"
	FileCopyPlanInlineContent   FileCopyPlanStrategy = "InlineContent"
	FileCopyPlanRemoteSource    FileCopyPlanStrategy = "RemoteSource"
	FileCopyPlanFileAsset       FileCopyPlanStrategy = "FileAsset"
	FileCopyPlanStringAsset     FileCopyPlanStrategy = "StringAsset"
	FileCopyPlanRemoteAsset     FileCopyPlanStrategy = "RemoteAsset"
	FileCopyPlanFileArchive     FileCopyPlanStrategy = "FileArchive"
	FileCopyPlanRemoteArchive   FileCopyPlanStrategy = "RemoteArchive"
	FileCopyPlanAssetArchive    FileCopyPlanStrategy = "AssetArchive"
)

type FileCopyPlan struct {
	Strategy  FileCopyPlanStrategy
	Hash      string
	Reader    io.ReadCloser
	Unarchive bool
}

func (r File) buildFileCopyPlan(inputs FileArgs) (FileCopyPlan, error) {
	if inputs.Content != nil {
		// FIXME: setting `content` in ansible.FileParameters breaks the Ansible
		// module because despite declaring `content` to be a valid parameter it
		// still expects and requires `src` to be set and exist.
		// return FileCopyPlan{
		// 	Strategy: FileCopyPlanInlineContent,
		// }, nil
		asset, err := passet.FromText(*inputs.Content)
		if err != nil {
			return FileCopyPlan{}, nil
		}
		inputs.Source = &ptypes.AssetOrArchive{
			Asset: asset,
		}
	}

	if inputs.RemoteSource != nil && *inputs.RemoteSource != "" {
		return FileCopyPlan{
			Strategy: FileCopyPlanRemoteSource,
		}, nil
	}

	if inputs.Source == nil {
		return FileCopyPlan{
			Strategy: FileCopyPlanNop,
		}, nil
	}

	plan := FileCopyPlan{}

	if inputs.Source.Asset != nil {
		asset := inputs.Source.Asset
		plan.Hash = asset.Hash
		switch {
		case asset.IsPath():
			plan.Strategy = FileCopyPlanFileAsset
		case asset.IsText():
			plan.Strategy = FileCopyPlanStringAsset
		case asset.IsURI():
			plan.Strategy = FileCopyPlanRemoteAsset
		default:
			if plan.Hash == "" {
				return plan, fmt.Errorf("unknown asset type: %#v", asset)
			}
			plan.Strategy = FileCopyPlanChecksumCompare
			return plan, nil
		}

		blob, err := asset.Read()
		if err != nil {
			return plan, err
		}

		plan.Reader = blob
	}

	if inputs.Source.Archive != nil {
		archive := inputs.Source.Archive
		plan.Hash = archive.Hash
		plan.Unarchive = true
		switch {
		case archive.IsPath():
			plan.Strategy = FileCopyPlanFileArchive
		case archive.IsURI():
			plan.Strategy = FileCopyPlanRemoteArchive
		case archive.IsAssets():
			plan.Strategy = FileCopyPlanAssetArchive
		default:
			if plan.Hash == "" {
				return plan, fmt.Errorf("unknown archive type: %#v", archive)
			}
			plan.Strategy = FileCopyPlanChecksumCompare
			return plan, nil
		}

		// FIXME: the TarGZIPArchive format is broken because the gzip.Writer is
		// never closed. This is an upstream Pulumi SDK bug.
		bbuf := bytes.Buffer{}
		zbuf := gzip.NewWriter(&bbuf)
		err := archive.Archive(parchive.TarArchive, zbuf)
		if err != nil {
			return plan, err
		}
		err = zbuf.Close()
		if err != nil {
			return plan, err
		}

		plan.Reader = io.NopCloser(&bbuf)
	}

	return plan, nil
}

func (r File) argsToFileTaskParameters(inputs FileArgs) (ansible.FileParameters, error) {
	var state *ansible.FileState
	if inputs.Ensure != nil {
		state = ansible.OptionalFileState(string(*inputs.Ensure))
	}
	return ansible.FileParameters{
		AccessTime:             inputs.AccessTime,
		AccessTimeFormat:       inputs.AccessTimeFormat,
		Attributes:             inputs.Attributes,
		Follow:                 inputs.Follow,
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
		Src:                    inputs.RemoteSource,
		State:                  state,
		UnsafeWrites:           inputs.UnsafeWrites,
	}, nil
}

func (r File) argsToCopyTaskParameters(inputs FileArgs) (ansible.CopyParameters, error) {
	params := ansible.CopyParameters{
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
		Selevel:       inputs.Selevel,
		Serole:        inputs.Serole,
		Setype:        inputs.Setype,
		Seuser:        inputs.Seuser,
		UnsafeWrites:  inputs.UnsafeWrites,
		Validate:      inputs.Validate,
	}
	return params, nil
}

func (r File) argsToStatTaskParameters(inputs FileArgs) (ansible.StatParameters, error) {
	return ansible.StatParameters{
		Follow: inputs.Follow,
		Path:   inputs.Path,
	}, nil
}

func (r File) updateState(inputs FileArgs, state FileState, changed bool) FileState {
	state.FileArgs = inputs
	state.Triggers = types.UpdateTriggerState(state.Triggers, inputs.Triggers, changed)
	return state
}

func (r File) Diff(ctx context.Context, req infer.DiffRequest[FileArgs, FileState]) (infer.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/resource/File.Diff", trace.WithAttributes(
		attribute.String("pulumi.operation", "diff"),
		attribute.String("pulumi.type", "mid:resource:File"),
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

	if req.Inputs.Path != req.State.Path {
		diff.HasChanges = true
		diff.DetailedDiff["path"] = p.PropertyDiff{
			Kind:      p.UpdateReplace,
			InputDiff: true,
		}
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(req.State, req.Inputs, []string{
			"accessTime",
			"accessTimeFormat",
			"attributes",
			"backup",
			"checksum",
			"content",
			"directoryMode",
			"ensure",
			"follow",
			"force",
			"group",
			"localFollow",
			"mode",
			"modificationTime",
			"modificationTimeFormat",
			"owner",
			"recurse",
			"remoteSource",
			"selevel",
			"serole",
			"setype",
			"seuser",
			"source",
			"unsafeWrites",
			"validate",
		}),
		types.DiffTriggers(req.State, req.Inputs),
	)

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(telemetry.OtelJSON("pulumi.diff", diff))
	return diff, nil
}

func (r File) runCreateUpdatePlay(
	ctx context.Context,
	inputs FileArgs,
	state FileState,
	dryRun bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.runCreateUpdatePlay", trace.WithAttributes(
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("inputs", inputs),
		attribute.Bool("dry_run", dryRun),
	))
	defer span.End()

	config := infer.GetConfig[types.Config](ctx)
	// several scenarios:
	// are we copying a local file to remote?
	//   - `copy` task
	// are we copying a remote file to remote?
	//   - `copy` task
	// are we setting the content of a remote file?
	//   - `copy` task
	// are we tweaking some metadata of an existing remote file?
	//   - `file` task
	// are we creating a symlink?
	//   - `file` task

	copyNeeded := ptr.AnyNonNils(
		inputs.Source,
		inputs.Content,
	)

	fileNeeded := ptr.AnyNonNils(
		inputs.AccessTime,
		inputs.AccessTimeFormat,
		inputs.ModificationTime,
		inputs.ModificationTimeFormat,
		inputs.Recurse,
		inputs.Ensure,
	)

	defer func() {
		span.SetAttributes(
			attribute.Bool("copy_needed", copyNeeded),
			attribute.Bool("file_needed", fileNeeded),
		)
	}()

	if executor.PreviewUnreachable(ctx, config.Connection, dryRun) {
		span.SetAttributes(attribute.Bool("unreachable", true))
		span.SetStatus(codes.Ok, "")
		state = r.updateState(inputs, state, true)
		return state, nil
	}

	changed := false

	stat, err := executor.CallAgent[rpc.FileStatArgs, rpc.FileStatResult](ctx, config.Connection, rpc.RPCCall[rpc.FileStatArgs]{
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

	if copyNeeded {
		remoteSource := false
		if inputs.RemoteSource != nil && *inputs.RemoteSource != "" {
			remoteSource = true
		}

		params, err := r.argsToCopyTaskParameters(inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		if remoteSource {
			params.Src = inputs.RemoteSource
			params.RemoteSrc = ptr.Of(true)
		} else {
			fileCopyPlan, err := r.buildFileCopyPlan(inputs)
			span.SetAttributes(
				attribute.String("file_copy_path.strategy", string(fileCopyPlan.Strategy)),
				attribute.Bool("file_copy_path.unarchive", fileCopyPlan.Unarchive),
			)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				return state, err
			}

			switch fileCopyPlan.Strategy {
			case FileCopyPlanInvalid:
				err = fmt.Errorf("invalid file copy plan: %#v", fileCopyPlan)
				span.SetStatus(codes.Error, err.Error())
				return state, err

			case FileCopyPlanNop, FileCopyPlanRemoteSource:
				break

			case FileCopyPlanInlineContent:
				params.Content = inputs.Content
				break

			case FileCopyPlanChecksumCompare:
				if !dryRun {
					err = fmt.Errorf("unable to update file due to the source not being loaded")
					span.SetStatus(codes.Error, err.Error())
					return state, err
				}

				if !stat.Result.Exists || stat.Result.SHA256Checksum == nil {
					changed = true
				}

				if fileCopyPlan.Hash != *stat.Result.SHA256Checksum {
					changed = true
				}
				goto fileNeeded

			default:
				if fileCopyPlan.Reader == nil {
					err = fmt.Errorf("unsupported file copy plan: %s", fileCopyPlan.Strategy)
					span.SetStatus(codes.Error, err.Error())
					return state, err
				}

				// TODO: only conditionally copy based on stat and checksum
				params.RemoteSrc = ptr.Of(true)
				stagedPath, err := executor.StageFile(ctx, config.Connection, fileCopyPlan.Reader)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					return state, err
				}

				if fileCopyPlan.Unarchive {
					result, err := executor.CallAgent[rpc.UntarArgs, rpc.UntarResult](ctx, config.Connection, rpc.RPCCall[rpc.UntarArgs]{
						RPCFunction: rpc.RPCUntar,
						Args: rpc.UntarArgs{
							SourceFilePath:  stagedPath,
							TargetDirectory: stagedPath + ".unarchived",
						},
					})
					if err != nil {
						return state, err
					}
					if result.Error != "" {
						return state, errors.New(result.Error)
					}
					params.Src = ptr.Of(stagedPath + ".unarchived")
				} else {
					params.Src = &stagedPath
				}
			}
		}

		if dryRun {
			// exit early if the parent dir doesn't exist
			dir := filepath.Dir(params.Dest)
			span.SetAttributes(attribute.String("parent_directory.path", dir))

			dirStat, err := executor.CallAgent[rpc.FileStatArgs, rpc.FileStatResult](ctx, config.Connection, rpc.RPCCall[rpc.FileStatArgs]{
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

			span.SetAttributes(attribute.Bool("parent_directory.exists", dirStat.Result.Exists))

			if !dirStat.Result.Exists {
				span.SetStatus(codes.Ok, "")
				state = r.updateState(inputs, state, true)
				return state, err
			}
		}

		result, err := executor.AnsibleExecute[
			ansible.CopyParameters,
			ansible.CopyReturn,
		](ctx, config.Connection, params, dryRun)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}

		if result.IsChanged() {
			changed = true
		}

		state.BackupFile = result.BackupFile
	}

fileNeeded:
	if fileNeeded {
		params, err := r.argsToFileTaskParameters(inputs)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		result, err := executor.AnsibleExecute[
			ansible.FileParameters,
			ansible.FileReturn,
		](ctx, config.Connection, params, dryRun)
		if err != nil {
			if !dryRun {
				span.SetStatus(codes.Error, err.Error())
				return state, err
			}
			changed = true
		}
		if result.IsChanged() {
			changed = true
		}
	}

	statParams, err := r.argsToStatTaskParameters(inputs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}
	statResult, err := executor.AnsibleExecute[
		ansible.StatParameters,
		ansible.StatReturn,
	](ctx, config.Connection, statParams, dryRun)
	if err != nil {
		if !dryRun {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		changed = true
	}
	if statResult.IsChanged() {
		changed = true
	}

	state.Stat, err = rpc.AnyToJSONT[FileStateStat](statResult.Stat)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state = r.updateState(inputs, state, changed)

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r File) Create(ctx context.Context, req infer.CreateRequest[FileArgs]) (infer.CreateResponse[FileState], error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Create", trace.WithAttributes(
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

	state, err = r.runCreateUpdatePlay(ctx, req.Inputs, state, req.DryRun)
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
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Read", trace.WithAttributes(
		attribute.String("pulumi.operation", "read"),
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.inputs", req.Inputs),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	state := req.State
	defer span.SetAttributes(telemetry.OtelJSON("pulumi.state", state))

	state, err := r.runCreateUpdatePlay(ctx, req.Inputs, state, true)
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
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Update", trace.WithAttributes(
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

	state, err := r.runCreateUpdatePlay(ctx, req.Inputs, state, req.DryRun)
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

func (r File) Delete(ctx context.Context, req infer.DeleteRequest[FileState]) (infer.DeleteResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Delete", trace.WithAttributes(
		attribute.String("pulumi.operation", "delete"),
		attribute.String("pulumi.type", "mid:resource:File"),
		attribute.String("pulumi.id", req.ID),
		telemetry.OtelJSON("pulumi.state", req.State),
	))
	defer span.End()

	shouldDelete := ptr.AnyNonNils(
		req.State.Source,
		req.State.Content,
		req.State.AccessTime,
		req.State.AccessTimeFormat,
		req.State.ModificationTime,
		req.State.ModificationTimeFormat,
		req.State.Recurse,
		req.State.Ensure,
	)

	span.SetAttributes(attribute.Bool("should_delete", shouldDelete))

	if !shouldDelete {
		span.SetStatus(codes.Ok, "")
		return infer.DeleteResponse{}, nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToFileTaskParameters(FileArgs{
		Path:   req.State.Path,
		Ensure: (*FileEnsure)(ptr.Of(string(FileEnsureAbsent))),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return infer.DeleteResponse{}, err
	}

	_, err = executor.AnsibleExecute[
		ansible.FileParameters,
		ansible.FileReturn,
	](ctx, config.Connection, parameters, false)
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
