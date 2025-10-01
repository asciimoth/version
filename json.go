package main

import (
	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/inplace/json"
)

func init() {
	RegisterSource("json", func() Source { return &JSONSource{} })
	RegisterDefaultSource("PackageJson", SourceWithMeta{
		VPrefix: VPrefixFalse,
		Source: &JSONSource{
			"package.json",
			[]string{"version"},
		},
	})
}

type JSONSource struct {
	Path    string
	KeyPath []string
}

func (d *JSONSource) IsCanBeLesser() bool {
	return false
}

func (d *JSONSource) IsReadOnly() bool {
	return false
}

func (d *JSONSource) Get(fs FS) (*semver.Version, error) {
	return getFromDoc(fs, json.New, d.KeyPath, d.Path)
}

func (d *JSONSource) Set(v semver.Version, fs FS) error {
	return setToDoc(v, fs, json.NewHuJSON, d.KeyPath, d.Path)
}
