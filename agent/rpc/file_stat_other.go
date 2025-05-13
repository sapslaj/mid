//go:build !linux

package rpc

import "errors"

func FileStat(args FileStatArgs) (FileStatResult, error) {
	return FileStatResult{}, errors.ErrUnsupported
}
