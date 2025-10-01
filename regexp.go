package main

import (
	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/inplace/regexp"
)

func init() {
	RegisterSource("regexp", func() Source { return &RegexpSource{} })
}

type RegexpSource struct {
	Path    string
	KeyPath []string
}

func (d *RegexpSource) IsCanBeLesser() bool {
	return false
}

func (d *RegexpSource) IsReadOnly() bool {
	return false
}

func (d *RegexpSource) Get(fs FS) (*semver.Version, error) {
	return getFromDoc(fs, regexp.New, d.KeyPath, d.Path)
}

func (d *RegexpSource) Set(v semver.Version, fs FS) error {
	return setToDoc(v, fs, regexp.New, d.KeyPath, d.Path)
}
