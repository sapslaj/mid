package midtypes

import (
	"strconv"
	"time"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
)

type FileStatFileMode struct {
	IsDir     bool   `pulumi:"isDir"`
	IsRegular bool   `pulumi:"isRegular"`
	Int       int    `pulumi:"int"`
	Octal     string `pulumi:"octal"`
	String    string `pulumi:"string"`
}

type FileStatState struct {
	Exists         bool              `pulumi:"exists"`
	BaseName       *string           `pulumi:"baseName,optional"`
	Size           *int64            `pulumi:"size,optional"`
	FileMode       *FileStatFileMode `pulumi:"fileMode,optional"`
	ModifiedTime   *string           `pulumi:"modifiedTime,optional"`
	AccessTime     *string           `pulumi:"accessTime,optional"`
	CreateTime     *string           `pulumi:"createTime,optional"`
	Dev            *int              `pulumi:"dev,optional"`
	Gid            *int              `pulumi:"gid,optional"`
	GroupName      *string           `pulumi:"groupName,optional"`
	Inode          *int              `pulumi:"inode,optional"`
	Nlink          *int              `pulumi:"nlink,optional"`
	Uid            *int              `pulumi:"uid,optional"`
	UserName       *string           `pulumi:"userName,optional"`
	SHA256Checksum *string           `pulumi:"sha256Checksum,optional"`
}

func FileStatStateFromRPCResult(result rpc.FileStatResult) FileStatState {
	output := FileStatState{
		Exists:         result.Exists,
		BaseName:       result.BaseName,
		Size:           result.Size,
		GroupName:      result.GroupName,
		UserName:       result.UserName,
		SHA256Checksum: result.SHA256Checksum,
	}
	if result.FileMode != nil {
		output.FileMode = &FileStatFileMode{
			IsDir:     result.FileMode.IsDir(),
			IsRegular: result.FileMode.IsRegular(),
			Int:       int(*result.FileMode),
			Octal:     strconv.FormatUint(uint64(*result.FileMode), 8),
			String:    result.FileMode.String(),
		}
	}
	if result.ModifiedTime != nil {
		output.ModifiedTime = ptr.Of(result.ModifiedTime.Format(time.RFC3339Nano))
	}
	if result.AccessTime != nil {
		output.AccessTime = ptr.Of(result.AccessTime.Format(time.RFC3339Nano))
	}
	if result.CreateTime != nil {
		output.CreateTime = ptr.Of(result.CreateTime.Format(time.RFC3339Nano))
	}
	if result.Dev != nil {
		output.Dev = ptr.Of(int(*result.Dev))
	}
	if result.Gid != nil {
		output.Gid = ptr.Of(int(*result.Gid))
	}
	if result.Inode != nil {
		output.Inode = ptr.Of(int(*result.Inode))
	}
	if result.Nlink != nil {
		output.Nlink = ptr.Of(int(*result.Nlink))
	}
	if result.Uid != nil {
		output.Uid = ptr.Of(int(*result.Uid))
	}

	return output
}
