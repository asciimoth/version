package main

import (
	"github.com/Masterminds/semver/v3"
)

func init() {
	RegisterSource("debug", func() Source { return &DebugSource{} })
}

type DebugSource struct {
	Version     string
	RO          bool
	CanBeLesser bool
	log         Log
}

func (d *DebugSource) IsCanBeLesser() bool {
	return d.CanBeLesser
}

func (d *DebugSource) IsReadOnly() bool {
	return d.RO
}

func (d *DebugSource) Get(_ FS) (*semver.Version, error) {
	if d.log != nil {
		d.log("asked for version, returning %s" + d.Version)
	}
	if d.Version == "" {
		return nil, nil //nolint:nilnil
	}
	return semver.NewVersion(d.Version)
}

func (d *DebugSource) Set(v semver.Version, _ FS) error {
	if d.log != nil {
		d.log(v.String() + " is set")
	}
	return nil
}
