package main

import (
	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/inplace/yaml"
)

func init() {
	RegisterSource("yaml", func() Source { return &YamlSource{} })
}

type YamlSource struct {
	Path    string
	KeyPath []string
}

func (d *YamlSource) IsCanBeLesser() bool {
	return false
}

func (d *YamlSource) IsReadOnly() bool {
	return false
}

func (d *YamlSource) Get(fs FS) (*semver.Version, error) {
	return getFromDoc(fs, yaml.New, d.KeyPath, d.Path)
}

func (d *YamlSource) Set(v semver.Version, fs FS) error {
	return setToDoc(v, fs, yaml.New, d.KeyPath, d.Path)
}
