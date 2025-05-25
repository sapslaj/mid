package resource

import (
	"context"
	"fmt"
	"path/filepath"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	ptypes "github.com/pulumi/pulumi-go-provider/infer/types"
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
	Path                   *string                `pulumi:"path,optional"`
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
	Path       string               `pulumi:"path"`
	BackupFile *string              `pulumi:"backupFile,optional"`
	Stat       FileStateStat        `pulumi:"stat"`
	Triggers   types.TriggersOutput `pulumi:"triggers"`
}

func (r File) argsToFileTaskParameters(input FileArgs) (*ansible.FileParameters, error) {
	var state *ansible.FileState
	if input.Ensure != nil {
		state = ansible.OptionalFileState(string(*input.Ensure))
	}
	return &ansible.FileParameters{
		AccessTime:             input.AccessTime,
		AccessTimeFormat:       input.AccessTimeFormat,
		Attributes:             input.Attributes,
		Follow:                 input.Follow,
		Group:                  input.Group,
		Mode:                   ptr.ToAny(input.Mode),
		ModificationTime:       input.ModificationTime,
		ModificationTimeFormat: input.ModificationTimeFormat,
		Owner:                  input.Owner,
		Path:                   *input.Path,
		Recurse:                input.Recurse,
		Selevel:                input.Selevel,
		Serole:                 input.Serole,
		Setype:                 input.Setype,
		Seuser:                 input.Seuser,
		Src:                    input.RemoteSource,
		State:                  state,
		UnsafeWrites:           input.UnsafeWrites,
	}, nil
}

func (r File) argsToSource(input FileArgs) (*string, error) {
	if input.RemoteSource != nil {
		return input.RemoteSource, nil
	} else if input.Source != nil {
		if input.Source.Asset != nil {
			if input.Source.Asset.Text != "" {
				return &input.Source.Asset.Text, nil
			} else if input.Source.Asset.Path != "" {
				abs, err := filepath.Abs(input.Source.Asset.Path)
				if err != nil {
					return nil, err
				}
				return &abs, nil
			}
		} else if input.Source.Archive != nil {
			abs, err := filepath.Abs(input.Source.Archive.Path)
			if err != nil {
				return nil, err
			}
			return &abs, nil
		}
	}
	return nil, nil
}

func (r File) argsToCopyTaskParameters(input FileArgs) (*ansible.CopyParameters, error) {
	isRemoteSource := input.RemoteSource != nil
	source, err := r.argsToSource(input)
	if err != nil {
		return nil, err
	}

	return &ansible.CopyParameters{
		Attributes:    input.Attributes,
		Backup:        input.Backup,
		Checksum:      input.Checksum,
		Content:       input.Content,
		Dest:          *input.Path,
		DirectoryMode: ptr.ToAny(input.DirectoryMode),
		Follow:        input.Follow,
		Force:         input.Force,
		Group:         input.Group,
		LocalFollow:   input.LocalFollow,
		Mode:          ptr.ToAny(input.Mode),
		Owner:         input.Owner,
		RemoteSrc:     ptr.Of(isRemoteSource),
		Selevel:       input.Selevel,
		Serole:        input.Serole,
		Setype:        input.Setype,
		Seuser:        input.Seuser,
		Src:           source,
		UnsafeWrites:  input.UnsafeWrites,
		Validate:      input.Validate,
	}, nil
}

func (r File) argsToStatTaskParameters(input FileArgs) (*ansible.StatParameters, error) {
	return &ansible.StatParameters{
		Follow: input.Follow,
		Path:   *input.Path,
	}, nil
}

func (r File) updateState(olds FileState, news FileArgs, changed bool) FileState {
	olds.FileArgs = news
	if news.Path != nil {
		olds.Path = *news.Path
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r File) Diff(
	ctx context.Context,
	id string,
	olds FileState,
	news FileArgs,
) (p.DiffResponse, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Diff", trace.WithAttributes(
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

	if news.Path == nil {
		news.Path = &olds.Path
	} else if *news.Path != olds.Path {
		diff.HasChanges = true
		diff.DetailedDiff["path"] = p.PropertyDiff{
			Kind:      p.UpdateReplace,
			InputDiff: true,
		}
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
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
			"path",
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
		types.DiffTriggers(olds, news),
	)

	return diff, nil
}

func (r File) runCreateUpdatePlay(
	ctx context.Context,
	state FileState,
	input FileArgs,
	preview bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.runCreateUpdatePlay", trace.WithAttributes(
		telemetry.OtelJSON("state", state),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
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

	tasks := []any{}

	copyTaskIndex := -1
	fileTaskIndex := -1
	statTaskIndex := -1

	copyNeeded := ptr.AnyNonNils(
		input.Source,
		input.Content,
	)

	fileNeeded := ptr.AnyNonNils(
		input.AccessTime,
		input.AccessTimeFormat,
		input.ModificationTime,
		input.ModificationTimeFormat,
		input.Recurse,
		input.Ensure,
	)

	defer func() {
		span.SetAttributes(
			attribute.Int("copy_task_index", copyTaskIndex),
			attribute.Int("file_task_index", fileTaskIndex),
			attribute.Int("stat_task_index", statTaskIndex),
			attribute.Bool("copy_needed", copyNeeded),
			attribute.Bool("file_needed", fileNeeded),
		)
	}()

	if preview && copyNeeded {
		source, err := r.argsToSource(input)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if source == nil {
			copyNeeded = false
			fileNeeded = true
		}
	}

	if copyNeeded {
		params, err := r.argsToCopyTaskParameters(input)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		copyTaskIndex = len(tasks)
		tasks = append(tasks, map[string]any{
			"ansible.builtin.copy": params,
		})
	}

	if fileNeeded {
		params, err := r.argsToFileTaskParameters(input)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		fileTaskIndex = len(tasks)
		tasks = append(tasks, map[string]any{
			"ansible.builtin.file": params,
			"ignore_errors":        copyNeeded || preview,
		})
	}

	statParams, err := r.argsToStatTaskParameters(input)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}
	statTaskIndex = len(tasks)
	tasks = append(tasks, map[string]any{
		"ansible.builtin.stat": statParams,
		"ignore_errors":        preview,
	})

	connectAttempts := 10
	if preview {
		connectAttempts = 4
	}
	canConnect, err := executor.CanConnect(ctx, config.Connection, connectAttempts)

	if !canConnect {
		if preview {
			return state, nil
		}

		if err == nil {
			err = fmt.Errorf("cannot connect to host")
		} else {
			err = fmt.Errorf("cannot connect to host: %w", err)
		}
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       preview,
		Tasks:       tasks,
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	changed := false

	if copyNeeded {
		result, err := executor.GetTaskResult[ansible.CopyReturn](output, 0, copyTaskIndex)
		if err != nil {
			return state, err
		}
		if result.IsChanged() {
			changed = true
		}
		state.BackupFile = result.BackupFile
	}

	if fileNeeded {
		result, err := executor.GetTaskResult[ansible.FileReturn](output, 0, fileTaskIndex)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return state, err
		}
		if result.IsChanged() {
			changed = true
		}
	}

	statResult, err := executor.GetTaskResult[ansible.StatReturn](output, 0, statTaskIndex)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state.Stat, err = rpc.AnyToJSONT[FileStateStat](statResult.Stat)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return state, err
	}

	state = r.updateState(state, input, changed)

	span.SetStatus(codes.Ok, "")
	return state, nil
}

func (r File) Create(
	ctx context.Context,
	name string,
	input FileArgs,
	preview bool,
) (string, FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Create", trace.WithAttributes(
		attribute.String("name", name),
		telemetry.OtelJSON("input", input),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	if input.Path == nil {
		input.Path = ptr.Of(name)
	}

	state := r.updateState(FileState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", state, err
	}
	span.SetAttributes(attribute.String("id", id))

	state, err = r.runCreateUpdatePlay(ctx, state, input, preview)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, state, nil
}

func (r File) Read(
	ctx context.Context,
	id string,
	inputs FileArgs,
	state FileState,
) (string, FileArgs, FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Read", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("inputs", inputs),
		telemetry.OtelJSON("state", state),
	))
	defer span.End()

	if inputs.Path == nil {
		inputs.Path = &state.Path
	}

	state, err := r.runCreateUpdatePlay(ctx, state, inputs, true)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return id, inputs, state, err
	}

	span.SetStatus(codes.Ok, "")
	return id, inputs, state, nil
}

func (r File) Update(
	ctx context.Context,
	id string,
	olds FileState,
	news FileArgs,
	preview bool,
) (FileState, error) {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Update", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("olds", olds),
		telemetry.OtelJSON("news", news),
		attribute.Bool("preview", preview),
	))
	defer span.End()

	if news.Path == nil {
		news.Path = &olds.Path
	}

	olds, err := r.runCreateUpdatePlay(ctx, olds, news, preview)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return olds, err
	}

	span.SetStatus(codes.Ok, "")
	return olds, nil
}

func (r File) Delete(
	ctx context.Context,
	id string,
	props FileState,
) error {
	ctx, span := Tracer.Start(ctx, "mid:resource:File.Delete", trace.WithAttributes(
		attribute.String("id", id),
		telemetry.OtelJSON("props", props),
	))
	defer span.End()

	shouldDelete := ptr.AnyNonNils(
		props.Source,
		props.Content,
		props.AccessTime,
		props.AccessTimeFormat,
		props.ModificationTime,
		props.ModificationTimeFormat,
		props.Recurse,
		props.Ensure,
	)

	span.SetAttributes(attribute.Bool("should_delete", shouldDelete))

	if !shouldDelete {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToFileTaskParameters(FileArgs{
		Path:   &props.Path,
		Ensure: (*FileEnsure)(ptr.Of(string(FileEnsureAbsent))),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection, 10)

	if !canConnect {
		if config.GetDeleteUnreachable() {
			return nil
		}

		if err == nil {
			err = fmt.Errorf("cannot connect to host")
		} else {
			err = fmt.Errorf("cannot connect to host: %w", err)
		}
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.file": parameters,
			},
		},
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
