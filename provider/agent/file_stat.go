package agent

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
)

type FileStat struct{}

type FileStatInputs struct {
	Path              string `pulumi:"path"`
	FollowSymlinks    bool   `pulumi:"followSymlinks,optional"`
	CalculateChecksum bool   `pulumi:"calculateChecksum,optional"`
}

type FileStatFileMode struct {
	IsDir     bool   `pulumi:"isDir"`
	IsRegular bool   `pulumi:"isRegular"`
	Int       int    `pulumi:"int"`
	Octal     string `pulumi:"octal"`
	String    string `pulumi:"string"`
}

type FileStatOutputs struct {
	FileStatInputs
	Path           string            `pulumi:"path"`
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

func (f FileStat) Call(ctx context.Context, input FileStatInputs) (FileStatOutputs, error) {
	ctx, span := Tracer.Start(ctx, "mid:agent:fileStat.Call", trace.WithAttributes(
		attribute.String("pulumi.function.token", "mid:agent:fileStat"),
		telemetry.OtelJSON("pulumi.function.inputs", input),
	))
	defer span.End()

	out, err := CallAgent[rpc.FileStatArgs, rpc.FileStatResult](ctx, rpc.RPCFileStat, rpc.FileStatArgs{
		Path:              input.Path,
		FollowSymlinks:    input.FollowSymlinks,
		CalculateChecksum: input.CalculateChecksum,
	})

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	outputs := FileStatOutputs{
		FileStatInputs: input,
		Path:           out.Path,
		Exists:         out.Exists,
		BaseName:       out.BaseName,
		Size:           out.Size,
		GroupName:      out.GroupName,
		UserName:       out.UserName,
		SHA256Checksum: out.SHA256Checksum,
	}

	if out.FileMode != nil {
		outputs.FileMode = &FileStatFileMode{
			IsDir:     out.FileMode.IsDir(),
			IsRegular: out.FileMode.IsRegular(),
			Int:       int(*out.FileMode),
			Octal:     strconv.FormatUint(uint64(*out.FileMode), 8),
			String:    out.FileMode.String(),
		}
	}
	if out.ModifiedTime != nil {
		outputs.ModifiedTime = ptr.Of(out.ModifiedTime.Format(time.RFC3339Nano))
	}
	if out.AccessTime != nil {
		outputs.AccessTime = ptr.Of(out.AccessTime.Format(time.RFC3339Nano))
	}
	if out.CreateTime != nil {
		outputs.CreateTime = ptr.Of(out.CreateTime.Format(time.RFC3339Nano))
	}
	if out.Dev != nil {
		outputs.Dev = ptr.Of(int(*out.Dev))
	}
	if out.Gid != nil {
		outputs.Gid = ptr.Of(int(*out.Gid))
	}
	if out.Inode != nil {
		outputs.Inode = ptr.Of(int(*out.Inode))
	}
	if out.Nlink != nil {
		outputs.Nlink = ptr.Of(int(*out.Nlink))
	}
	if out.Uid != nil {
		outputs.Uid = ptr.Of(int(*out.Uid))
	}

	span.SetAttributes(telemetry.OtelJSON("pulumi.function.outputs", outputs))

	return outputs, err
}
