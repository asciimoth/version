package main

import "github.com/Masterminds/semver/v3"

type Source interface {
	IsCanBeLesser() bool
	IsReadOnly() bool
	Get(fs FS) (*semver.Version, error)
	Set(v semver.Version, fs FS) error
}

// Type -> default constructor.
var sources = map[string]func() Source{}

// Name -> Source.
var defaultSources = map[string]SourceWithMeta{}

func RegisterSource(srcType string, constructor func() Source) {
	sources[srcType] = constructor
}

func RegisterDefaultSource(name string, src SourceWithMeta) {
	defaultSources[name] = src
}
