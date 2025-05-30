package agent

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/pkg/telemetry"
)

type FileStat struct{}

type FileStatInput struct {
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

type FileStatOutput struct {
	FileStatInput
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

func (f FileStat) Invoke(
	ctx context.Context,
	req infer.FunctionRequest[FileStatInput],
) (infer.FunctionResponse[FileStatOutput], error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/agent/fileStat.Call", trace.WithAttributes(
		attribute.String("pulumi.function", "mid:agent:fileStat"),
		telemetry.OtelJSON("pulumi.input", req.Input),
	))
	defer span.End()

	out, err := CallAgent[rpc.FileStatArgs, rpc.FileStatResult](ctx, rpc.RPCFileStat, rpc.FileStatArgs{
		Path:              req.Input.Path,
		FollowSymlinks:    req.Input.FollowSymlinks,
		CalculateChecksum: req.Input.CalculateChecksum,
	})

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	output := FileStatOutput{
		FileStatInput:  req.Input,
		Path:           out.Path,
		Exists:         out.Exists,
		BaseName:       out.BaseName,
		Size:           out.Size,
		GroupName:      out.GroupName,
		UserName:       out.UserName,
		SHA256Checksum: out.SHA256Checksum,
	}

	if out.FileMode != nil {
		output.FileMode = &FileStatFileMode{
			IsDir:     out.FileMode.IsDir(),
			IsRegular: out.FileMode.IsRegular(),
			Int:       int(*out.FileMode),
			Octal:     strconv.FormatUint(uint64(*out.FileMode), 8),
			String:    out.FileMode.String(),
		}
	}
	if out.ModifiedTime != nil {
		output.ModifiedTime = ptr.Of(out.ModifiedTime.Format(time.RFC3339Nano))
	}
	if out.AccessTime != nil {
		output.AccessTime = ptr.Of(out.AccessTime.Format(time.RFC3339Nano))
	}
	if out.CreateTime != nil {
		output.CreateTime = ptr.Of(out.CreateTime.Format(time.RFC3339Nano))
	}
	if out.Dev != nil {
		output.Dev = ptr.Of(int(*out.Dev))
	}
	if out.Gid != nil {
		output.Gid = ptr.Of(int(*out.Gid))
	}
	if out.Inode != nil {
		output.Inode = ptr.Of(int(*out.Inode))
	}
	if out.Nlink != nil {
		output.Nlink = ptr.Of(int(*out.Nlink))
	}
	if out.Uid != nil {
		output.Uid = ptr.Of(int(*out.Uid))
	}

	span.SetAttributes(telemetry.OtelJSON("pulumi.output", output))

	return infer.FunctionResponse[FileStatOutput]{
		Output: output,
	}, err
}
