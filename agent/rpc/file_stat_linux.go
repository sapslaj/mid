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
		result.BaseName = fileInfo.Name()
		result.FileMode = fileInfo.Mode()
		result.ModifiedTime = fileInfo.ModTime()
		result.Size = fileInfo.Size()
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
	case syscall.Stat_t:
		result.AccessTime = time.Unix(stat.Atim.Unix())
		result.CreateTime = time.Unix(stat.Ctim.Unix())
		result.Dev = stat.Dev
		result.Gid = uint64(stat.Gid)
		grp, err := user.LookupGroupId(fmt.Sprint(stat.Gid))
		if err == nil && grp != nil {
			result.GroupName = grp.Name
		}
		result.Inode = stat.Ino
		result.Nlink = uint64(stat.Nlink)
		result.Uid = uint64(stat.Uid)
		usr, err := user.LookupId(fmt.Sprint(stat.Uid))
		if err == nil && usr != nil {
			result.UserName = usr.Name
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
		result.SHA256Checksum = fmt.Sprintf("%x", h.Sum(nil))
	}

	return result, nil
}
