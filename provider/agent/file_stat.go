package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/midtypes"
)

type FileStat struct{}

type FileStatInput struct {
	Path              string                   `pulumi:"path"`
	FollowSymlinks    bool                     `pulumi:"followSymlinks,optional"`
	CalculateChecksum bool                     `pulumi:"calculateChecksum,optional"`
	Connection        *midtypes.Connection     `pulumi:"connection,optional"`
	Config            *midtypes.ResourceConfig `pulumi:"config,optional"`
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
	midtypes.FileStatState
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

	out, err := CallAgent[rpc.FileStatArgs, rpc.FileStatResult](
		ctx,
		req.Input.Connection,
		req.Input.Config,
		rpc.RPCFileStat,
		rpc.FileStatArgs{
			Path:              req.Input.Path,
			FollowSymlinks:    req.Input.FollowSymlinks,
			CalculateChecksum: req.Input.CalculateChecksum,
		},
	)

	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	output := FileStatOutput{
		FileStatInput: req.Input,
		FileStatState: midtypes.FileStatStateFromRPCResult(out),
	}

	span.SetAttributes(telemetry.OtelJSON("pulumi.output", output))

	return infer.FunctionResponse[FileStatOutput]{
		Output: output,
	}, err
}
