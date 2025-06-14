package rpc

import (
	"os"

	"github.com/sapslaj/mid/agent/untar"
)

type UntarArgs struct {
	SourceFilePath  string
	TargetDirectory string
}

type UntarResult struct{}

func Untar(args UntarArgs) (UntarResult, error) {
	reader, err := os.Open(args.SourceFilePath)
	if err != nil {
		return UntarResult{}, err
	}
	err = untar.Untar(reader, args.TargetDirectory)
	if err != nil {
		return UntarResult{}, err
	}
	return UntarResult{}, nil
}
