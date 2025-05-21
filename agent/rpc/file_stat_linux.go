//go:build linux

package rpc

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"syscall"
	"time"

	"github.com/sapslaj/mid/pkg/ptr"
)

func FileStat(args FileStatArgs) (FileStatResult, error) {
	result := FileStatResult{
		Path: args.Path,
	}

	var fileInfo os.FileInfo
	var err error
	if args.FollowSymlinks {
		fileInfo, err = os.Stat(args.Path)
	} else {
		fileInfo, err = os.Lstat(args.Path)
	}

	if fileInfo != nil {
		result.BaseName = ptr.Of(fileInfo.Name())
		result.FileMode = ptr.Of(fileInfo.Mode())
		result.ModifiedTime = ptr.Of(fileInfo.ModTime())
		result.Size = ptr.Of(fileInfo.Size())
	}

	result.Exists = true

	if errors.Is(err, fs.ErrNotExist) {
		result.Exists = false
		return result, nil
	}

	if err != nil {
		return result, err
	}

	switch stat := fileInfo.Sys().(type) {
	case *syscall.Stat_t:
		result.AccessTime = ptr.Of(time.Unix(stat.Atim.Unix()))
		result.CreateTime = ptr.Of(time.Unix(stat.Ctim.Unix()))
		result.Dev = ptr.Of(stat.Dev)
		result.Gid = ptr.Of(uint64(stat.Gid))
		grp, err := user.LookupGroupId(fmt.Sprint(stat.Gid))
		if err == nil && grp != nil {
			result.GroupName = ptr.Of(grp.Name)
		}
		result.Inode = ptr.Of(stat.Ino)
		result.Nlink = ptr.Of(uint64(stat.Nlink))
		result.Uid = ptr.Of(uint64(stat.Uid))
		usr, err := user.LookupId(fmt.Sprint(stat.Uid))
		if err == nil && usr != nil {
			result.UserName = ptr.Of(usr.Name)
		}
	}

	if args.CalculateChecksum && !fileInfo.IsDir() {
		f, err := os.Open(args.Path)
		if err != nil {
			return result, err
		}
		defer f.Close()
		h := sha256.New()
		_, err = io.Copy(h, f)
		if err != nil {
			return result, err
		}
		result.SHA256Checksum = ptr.Of(fmt.Sprintf("%x", h.Sum(nil)))
	}

	return result, nil
}
