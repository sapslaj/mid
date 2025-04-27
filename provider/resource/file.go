package resource

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/aws/smithy-go/ptr"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	ptypes "github.com/pulumi/pulumi-go-provider/infer/types"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

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

type fileTaskParameters struct {
	AccessTime             *string `json:"access_time,omitempty"`
	AccessTimeFormat       *string `json:"access_time_format,omitempty"`
	Attributes             *string `json:"attributes,omitempty"`
	Follow                 *bool   `json:"follow,omitempty"`
	Force                  *bool   `json:"force,omitempty"`
	Group                  *string `json:"group,omitempty"`
	Mode                   *string `json:"mode,omitempty"`
	ModificationTime       *string `json:"modification_time,omitempty"`
	ModificationTimeFormat *string `json:"modification_time_format,omitempty"`
	Owner                  *string `json:"owner,omitempty"`
	Path                   string  `json:"path"`
	Recurse                *bool   `json:"recurse,omitempty"`
	Selevel                *string `json:"selevel,omitempty"`
	Serole                 *string `json:"serole,omitempty"`
	Setype                 *string `json:"setype,omitempty"`
	Seuser                 *string `json:"seuser,omitempty"`
	Src                    *string `json:"src,omitempty"`
	State                  *string `json:"state,omitempty"`
	UnsafeWrites           *bool   `json:"unsafe_writes,omitempty"`
}

type fileTaskResult struct {
	Changed *bool  `json:"changed,omitempty"`
	Diff    *any   `json:"diff,omitempty"`
	Path    string `json:"string"`
}

func (result *fileTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r File) argsToFileTaskParameters(input FileArgs) (*fileTaskParameters, error) {
	return &fileTaskParameters{
		AccessTime:             input.AccessTime,
		AccessTimeFormat:       input.AccessTimeFormat,
		Attributes:             input.Attributes,
		Follow:                 input.Follow,
		Group:                  input.Group,
		Mode:                   input.Mode,
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
		State:                  (*string)(input.Ensure),
		UnsafeWrites:           input.UnsafeWrites,
	}, nil
}

type copyTaskParameters struct {
	Attributes    *string `json:"attributes,omitempty"`
	Backup        *bool   `json:"backup,omitempty"`
	Checksum      *string `json:"checksum,omitempty"`
	Content       *string `json:"content,omitempty"`
	Dest          *string `json:"dest,omitempty"`
	DirectoryMode *string `json:"directory_mode,omitempty"`
	Follow        *bool   `json:"follow,omitempty"`
	Force         *bool   `json:"force,omitempty"`
	Group         *string `json:"group,omitempty"`
	LocalFollow   *bool   `json:"local_follow,omitempty"`
	Mode          *string `json:"mode,omitempty"`
	Owner         *string `json:"owner,omitempty"`
	RemoteSrc     *bool   `json:"remote_src,omitempty"`
	Selevel       *string `json:"selevel,omitempty"`
	Serole        *string `json:"serole,omitempty"`
	Setype        *string `json:"setype,omitempty"`
	Seuser        *string `json:"seuser,omitempty"`
	Src           *string `json:"src,omitempty"`
	UnsafeWrites  *bool   `json:"unsafe_writes,omitempty"`
	Validate      *string `json:"validate,omitempty"`
}

type copyTaskResult struct {
	Changed    *bool  `json:"changed,omitempty"`
	Diff       *any   `json:"diff,omitempty"`
	BackupFile string `json:"backup_file"`
	Checksum   string `json:"checksum"`
	Dest       string `json:"dest"`
	Gid        int    `json:"gid"`
	Group      string `json:"group"`
	Mode       string `json:"mode"`
	Owner      string `json:"owner"`
	Size       int    `json:"size"`
	Src        string `json:"src"`
	State      string `json:"state"`
	Uid        int    `json:"uid"`
}

func (result *copyTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
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

func (r File) argsToCopyTaskParameters(input FileArgs) (*copyTaskParameters, error) {
	isRemoteSource := input.RemoteSource != nil
	source, err := r.argsToSource(input)
	if err != nil {
		return nil, err
	}

	return &copyTaskParameters{
		Attributes:    input.Attributes,
		Backup:        input.Backup,
		Checksum:      input.Checksum,
		Content:       input.Content,
		Dest:          input.Path,
		DirectoryMode: input.DirectoryMode,
		Follow:        input.Follow,
		Force:         input.Force,
		Group:         input.Group,
		LocalFollow:   input.LocalFollow,
		Mode:          input.Mode,
		Owner:         input.Owner,
		RemoteSrc:     ptr.Bool(isRemoteSource),
		Selevel:       input.Selevel,
		Serole:        input.Serole,
		Setype:        input.Setype,
		Seuser:        input.Seuser,
		Src:           source,
		UnsafeWrites:  input.UnsafeWrites,
		Validate:      input.Validate,
	}, nil
}

type statTaskParameters struct {
	ChecksumAlgorithm *string `json:"checksum_algorithm,omitempty"`
	Follow            *bool   `json:"follow,omitempty"`
	GetAttributes     *bool   `json:"get_attributes,omitempty"`
	GetChecksum       *bool   `json:"get_checksum,omitempty"`
	GetMime           *bool   `json:"get_mime,omitempty"`
	Path              string  `json:"path"`
}

type statTaskResult struct {
	Stat FileStateStat `json:"stat"`
}

func (r File) argsToStatTaskParameters(input FileArgs) (*statTaskParameters, error) {
	return &statTaskParameters{
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

func anyNonNils(vs ...any) bool {
	for _, v := range vs {
		if v != nil && !reflect.ValueOf(v).IsNil() {
			return true
		}
	}
	return false
}

func (r File) Diff(
	ctx context.Context,
	id string,
	olds FileState,
	news FileArgs,
) (p.DiffResponse, error) {
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

	copyNeeded := anyNonNils(
		input.Source,
		input.Content,
	)

	fileNeeded := anyNonNils(
		input.AccessTime,
		input.AccessTimeFormat,
		input.ModificationTime,
		input.ModificationTimeFormat,
		input.Recurse,
		input.Ensure,
	)

	if preview && copyNeeded {
		source, err := r.argsToSource(input)
		if err != nil {
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
			return state, err
		}
		fileTaskIndex = len(tasks)
		tasks = append(tasks, map[string]any{
			"ansible.builtin.file": params,
			"ignore_errors":        copyNeeded && preview,
		})
	}

	statParams, err := r.argsToStatTaskParameters(input)
	if err != nil {
		return state, err
	}
	statTaskIndex = len(tasks)
	tasks = append(tasks, map[string]any{
		"ansible.builtin.stat": statParams,
		"ignore_errors":        preview,
	})

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: false,
		Become:      true,
		Check:       preview,
		Tasks:       tasks,
	})
	if err != nil {
		return state, err
	}

	changed := false

	if copyNeeded {
		result, err := executor.GetTaskResult[copyTaskResult](output, 0, copyTaskIndex)
		if err != nil {
			return state, err
		}
		if result.IsChanged() {
			changed = true
		}
		state.BackupFile = &result.BackupFile
	}

	if fileNeeded {
		result, err := executor.GetTaskResult[fileTaskResult](output, 0, fileTaskIndex)
		if err != nil {
			return state, err
		}
		if result.IsChanged() {
			changed = true
		}
	}

	statResult, err := executor.GetTaskResult[statTaskResult](output, 0, statTaskIndex)
	if err != nil {
		return state, err
	}

	state.Stat = statResult.Stat

	state = r.updateState(state, input, changed)

	return state, nil
}

func (r File) Create(
	ctx context.Context,
	name string,
	input FileArgs,
	preview bool,
) (string, FileState, error) {
	if input.Path == nil {
		input.Path = ptr.String(name)
	}

	state := r.updateState(FileState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	state, err = r.runCreateUpdatePlay(ctx, state, input, preview)
	if err != nil {
		return id, state, err
	}

	return id, state, nil
}

func (r File) Read(
	ctx context.Context,
	id string,
	inputs FileArgs,
	state FileState,
) (string, FileArgs, FileState, error) {
	if inputs.Path == nil {
		inputs.Path = &state.Path
	}

	state, err := r.runCreateUpdatePlay(ctx, state, inputs, true)
	if err != nil {
		return id, inputs, state, err
	}

	return id, inputs, state, nil
}

func (r File) Update(
	ctx context.Context,
	id string,
	olds FileState,
	news FileArgs,
	preview bool,
) (FileState, error) {
	if news.Path == nil {
		news.Path = &olds.Path
	}

	olds, err := r.runCreateUpdatePlay(ctx, olds, news, preview)
	if err != nil {
		return olds, err
	}

	return olds, nil
}

func (r File) Delete(
	ctx context.Context,
	id string,
	props FileState,
) error {
	shouldDelete := anyNonNils(
		props.Source,
		props.Content,
		props.AccessTime,
		props.AccessTimeFormat,
		props.ModificationTime,
		props.ModificationTimeFormat,
		props.Recurse,
		props.Ensure,
	)

	if !shouldDelete {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	parameters, err := r.argsToFileTaskParameters(FileArgs{
		Path:   &props.Path,
		Ensure: (*FileEnsure)(ptr.String(string(FileEnsureAbsent))),
	})
	if err != nil {
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

	return err
}
