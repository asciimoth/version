package main

import (
	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/inplace/toml"
)

func init() {
	RegisterSource("toml", func() Source { return &TOMLSource{} })
	RegisterDefaultSource("PyProject", SourceWithMeta{
		VPrefix: VPrefixFalse,
		Source: &TOMLSource{
			"pyproject.toml",
			[]string{"project", "version"},
		},
	})
	RegisterDefaultSource("Cargo", SourceWithMeta{
		VPrefix: VPrefixFalse,
		Source: &TOMLSource{
			"Cargo.toml",
			[]string{"package", "version"},
		},
	})
}

type TOMLSource struct {
	Path    string
	KeyPath []string
}

func (d *TOMLSource) IsCanBeLesser() bool {
	return false
}

func (d *TOMLSource) IsReadOnly() bool {
	return false
}

func (d *TOMLSource) Get(fs FS) (*semver.Version, error) {
	return getFromDoc(fs, toml.New, d.KeyPath, d.Path)
}

func (d *TOMLSource) Set(v semver.Version, fs FS) error {
	return setToDoc(v, fs, toml.New, d.KeyPath, d.Path)
}
