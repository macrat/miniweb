package main

import (
	"io/fs"
)

type FileType interface {
	Serve(*Context, fs.File, fs.FileInfo) error
}
