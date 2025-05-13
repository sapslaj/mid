package rpc

import (
	"io/fs"
	"time"
)

type FileStatArgs struct {
	Path              string
	FollowSymlinks    bool
	CalculateChecksum bool
}

type FileStatResult struct {
	// always returned
	Path   string
	Exists bool

	// usually returned
	BaseName     string
	Size         int64
	FileMode     fs.FileMode
	ModifiedTime time.Time

	// depends on FileInfo.Sys()
	AccessTime time.Time
	CreateTime time.Time
	Dev        uint64
	Gid        uint64
	GroupName  string
	Inode      uint64
	Nlink      uint64
	Uid        uint64
	UserName   string

	// conditional on CalculateChecksum
	SHA256Checksum string
}

// NOTE: `FileStat` implementation is in `file_stat_linux.go`. It needed to be
// build-flagged away due to "syscall" package weirdness
