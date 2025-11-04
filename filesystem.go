package main

import (
	"io/fs"
	"os"
)

type FileSystem interface {
	fs.FS
	WriteFile(name string, data []byte, perm fs.FileMode) error
	Remove(name string) error
	ReadDir(name string) ([]fs.DirEntry, error)
}

type OSFileSystem struct {
	dir string
}

func NewOSFileSystem(dir string) *OSFileSystem {
	return &OSFileSystem{dir: dir}
}

func (o *OSFileSystem) Open(name string) (fs.File, error) {
	return os.Open(o.dir + "/" + name)
}

func (o *OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(o.dir + "/" + name)
}

func (o *OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(o.dir+"/"+name, data, perm)
}

func (o *OSFileSystem) Remove(name string) error {
	return os.Remove(o.dir + "/" + name)
}
