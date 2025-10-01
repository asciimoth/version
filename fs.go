package main

import (
	"io/fs"
	"os"
	"path"
)

type FS interface {
	Open(path string) (*os.File, error)
	Rename(oldpath, newpath string) error
	Stat(path string) (os.FileInfo, error)
	Remove(path string) error
	OpenFile(path string, flag int, perm os.FileMode) (*os.File, error)
	Glob(pattern string) (matches []string, err error)
}

func FSFromRoot(root *os.Root) FS { //nolint:ireturn
	if root == nil {
		return nil
	}
	return &osRootFS{root: *root}
}

type osRootFS struct {
	root os.Root
}

func (osr *osRootFS) Open(path string) (*os.File, error) {
	return osr.root.Open(path)
}

func (osr *osRootFS) Rename(oldpath, newpath string) error {
	return osr.root.Rename(oldpath, newpath)
}

func (osr *osRootFS) Stat(path string) (os.FileInfo, error) {
	return osr.root.Stat(path)
}

func (osr *osRootFS) Remove(path string) error {
	return osr.root.Remove(path)
}

func (osr *osRootFS) OpenFile(
	path string,
	flag int,
	perm os.FileMode,
) (*os.File, error) {
	return osr.root.OpenFile(path, flag, perm)
}

func (osr *osRootFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(osr.root.FS(), pattern)
}

type filteredFS struct {
	fs       FS
	patterns []string
}

func (f *filteredFS) Open(path string) (*os.File, error) {
	return f.fs.Open(path)
}

func (f *filteredFS) Rename(oldpath, newpath string) error {
	return f.fs.Rename(oldpath, newpath)
}

func (f *filteredFS) Stat(path string) (os.FileInfo, error) {
	return f.fs.Stat(path)
}

func (f *filteredFS) Remove(path string) error {
	return f.fs.Remove(path)
}

func (f *filteredFS) OpenFile(
	path string,
	flag int,
	perm os.FileMode,
) (*os.File, error) {
	return f.fs.OpenFile(path, flag, perm)
}

func (f *filteredFS) Glob(pattern string) ([]string, error) {
	m, err := f.fs.Glob(pattern)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(m))
	for _, m := range m {
		skip := false
		for _, p := range f.patterns {
			t, err := path.Match(p, m)
			if err != nil || t {
				skip = true
				continue
			}
		}
		if skip {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}
