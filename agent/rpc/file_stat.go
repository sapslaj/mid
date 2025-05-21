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
	BaseName     *string      `json:",omitempty"`
	Size         *int64       `json:",omitempty"`
	FileMode     *fs.FileMode `json:",omitempty"`
	ModifiedTime *time.Time   `json:",omitempty"`

	// depends on FileInfo.Sys()
	AccessTime *time.Time `json:",omitempty"`
	CreateTime *time.Time `json:",omitempty"`
	Dev        *uint64    `json:",omitempty"`
	Gid        *uint64    `json:",omitempty"`
	GroupName  *string    `json:",omitempty"`
	Inode      *uint64    `json:",omitempty"`
	Nlink      *uint64    `json:",omitempty"`
	Uid        *uint64    `json:",omitempty"`
	UserName   *string    `json:",omitempty"`

	// conditional on CalculateChecksum
	SHA256Checksum *string `json:",omitempty"`
}

// NOTE: `FileStat` implementation is in `file_stat_linux.go`. It needed to be
// build-flagged away due to "syscall" package weirdness
