package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/rewrite"
	"github.com/pelletier/go-toml"
)

var errSubtreeNotFound = errors.New("subtree not found")

type Log = func(string)

type VPrefixMode = int

const (
	VPrefixAuto VPrefixMode = iota
	VPrefixTrue
	VPrefixFalse
)

func NewVPrefixMode(name string) VPrefixMode {
	if strings.EqualFold(name, "True") {
		return VPrefixTrue
	}
	if strings.EqualFold(name, "False") {
		return VPrefixFalse
	}
	return VPrefixAuto
}

type SourceWithMeta struct {
	VPrefix  VPrefixMode
	Disabled bool
	Source   Source
}

func (swm *SourceWithMeta) UnmarshalTOML(data any) error {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf(
			"expected table for source, got %T",
			data,
		) //nolint:err113
	}

	// find type key case-insensitively
	var typ string
	for k, v := range m {
		if strings.EqualFold(k, "Type") {
			if s, ok := v.(string); ok {
				typ = s
			} else {
				return fmt.Errorf("type field is not a string (got %T)", v) //nolint:err113
			}
			break
		}
	}
	if typ == "" {
		return errors.New("missing type field for source") //nolint:err113
	}

	// extract VPrefix if present (case-insensitive key match)
	for k, v := range m {
		if strings.EqualFold(k, "VPrefix") {
			switch val := v.(type) {
			case string:
				swm.VPrefix = NewVPrefixMode(val)
			case bool:
				if val {
					swm.VPrefix = VPrefixTrue
				} else {
					swm.VPrefix = VPrefixFalse
				}
			case int64:
				swm.VPrefix = VPrefixMode(val)
			case int:
				swm.VPrefix = val
			case float64:
				swm.VPrefix = int(val)
			}
			break
		} else if strings.EqualFold(k, "Disabled") {
			if strings.EqualFold(fmt.Sprint(v), "true") {
				swm.Disabled = true
			}
		}
	}

	constructor := sources[typ]
	if constructor == nil {
		return fmt.Errorf(
			"there is no registered constructor for source type %s",
			typ,
		)
	}
	instance := constructor()

	// Marshal the parsed map back to TOML bytes, then unmarshal into the concrete instance.
	// This keeps decoding behavior consistent with the toml tags / rules.
	b, err := toml.Marshal(m)
	if err != nil {
		return err
	}
	if err := toml.Unmarshal(b, instance); err != nil {
		return err
	}

	swm.Source = instance
	return nil
}

type Name = string

type report struct {
	v *semver.Version
	s SourceWithMeta
	n Name
}

type SourceGroup struct {
	DefaultVersion  string
	Sources         map[Name]SourceWithMeta
	Strict          bool
	Trace, Log, Err Log
	getFS           FS
	setFS           FS
	IgnoredFiles    []string
	ReadOnlyFiles   []string
}

func NewGroupSource(
	defaultVersion string,
	sources map[Name]SourceWithMeta,
	strict bool,
	trace, log, elog Log,
	fs FS,
	ignoredFiles, roFiles []string,
) (*SourceGroup, error) {
	ifs := &filteredFS{fs, ignoredFiles}
	rofs := &filteredFS{ifs, roFiles}
	gs := &SourceGroup{
		defaultVersion,
		sources,
		strict,
		trace, log, elog,
		ifs, rofs,
		ignoredFiles, roFiles,
	}
	err := gs.verify()
	if err != nil {
		return nil, err
	}
	return gs, nil
}

func GroupFromConfig(
	fs FS,
	trace, log, errLog Log,
	strict bool,
) (*SourceGroup, error) {
	files := []struct {
		filename string
		subtree  string
	}{
		{"version.toml", ""},
		{".version.toml", ""},
		{"pyproject.toml", "tool.version"},
	}
	for _, file := range files {
		bytes, err := rewrite.Read(fs, file.filename)
		if err != nil {
			continue
		}
		gr, err := GroupFromToml(bytes, trace, log, errLog, fs, file.subtree)
		if err != nil {
			if errors.Is(err, errSubtreeNotFound) {
				continue
			}
		}
		if strict {
			gr.Strict = strict
		}
		return gr, nil
	}
	return NewGroupSource(
		"",
		defaultSources,
		strict,
		func(_ string) {},
		log,
		errLog,
		fs,
		[]string{},
		[]string{},
	)
}

func GroupFromToml(
	data []byte,
	trace, log, errLog Log,
	fs FS,
	subtree string,
) (*SourceGroup, error) {
	tree, err := toml.LoadBytes(data)
	if err != nil {
		return nil, err
	}
	stree := tree.Get(subtree)
	if stree == nil {
		return nil, errSubtreeNotFound
	}
	if sub, ok := stree.(*toml.Tree); ok {
		tree = sub
	}
	var gs SourceGroup
	if err := tree.Unmarshal(&gs); err != nil {
		return nil, err
	}

	ifs := &filteredFS{fs, gs.IgnoredFiles}
	rofs := &filteredFS{ifs, gs.ReadOnlyFiles}
	gs.Trace = trace
	gs.Log = log
	gs.Err = errLog
	gs.getFS = ifs
	gs.setFS = rofs
	err = gs.verify()
	if err != nil {
		return nil, err
	}
	return &gs, nil
}

func (g *SourceGroup) Filter(names []Name) map[Name]SourceWithMeta {
	srcs := make(map[Name]SourceWithMeta)
	for name, src := range g.Sources {
		if !slices.Contains(names, "none") && slices.Contains(names, name) {
			srcs[name] = src
		}
	}
	return srcs
}

func (g *SourceGroup) Fetch( //nolint:nonamedreturns
	names []Name,
) (reports []report, vp bool, err error) {
	vp = false
	reports = []report{}
	sources := g.Sources
	if len(names) > 0 {
		sources = g.Filter(names)
	}
	for name, src := range sources {
		if src.Disabled {
			g.Trace(fmt.Sprintf("  %s skipped as disabled", name))
			continue
		}
		v, e := src.Source.Get(g.getFS)
		if e != nil {
			g.Err(fmt.Sprintf("  %s failed with: %s", name, e))
			if err == nil {
				err = e
			}
			continue
		}
		vp = vp || hasVPrefix(v)
		// For some reasons sometimes semver.NewVersion rurns nil for
		// both version and error
		reports = append(reports, report{v, src, name})
	}
	if err != nil {
		return
	}
	// Sorting reporst for better log messages later
	slices.SortFunc(reports, func(a, b report) int {
		if b.v == nil {
			if a.v == nil {
				return 0
			}
			return -1
		}
		if a.v == nil {
			return 1
		}
		return b.v.Compare(a.v)
	})
	return
}

func (g *SourceGroup) GetMax(names []Name, versions []semver.Version) (
	*semver.Version,
	error,
) {
	var err error
	var version *semver.Version
	reports := []report{}
	if len(names) > 0 {
		reports, _, err = g.Fetch(names)
	}
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if version == nil {
			version = &v
			continue
		}
		if v.GreaterThan(version) {
			version = &v
		}
	}
	for _, r := range reports {
		if version == nil {
			version = r.v
		}
		if r.v != nil && r.v.GreaterThan(version) {
			version = r.v
		}
	}
	if version == nil {
		dv := g.DefaultVersion
		if dv == "" {
			dv = "0.1.0"
		}
		v, err := semver.NewVersion(dv)
		if err != nil {
			return nil, err
		}
		version = v
	}
	return version, nil
}

// Returns only first error.
// If any of sources reports version with leading v, result have leading v too.
func (g *SourceGroup) Get(names []Name) (version *semver.Version, err error) {
	g.Log("fetching versions from sources...")
	// Collecting reports from all Sources
	reports, vp, err := g.Fetch(names)
	if err != nil {
		return
	}
	// Verifying that reported versions are matching
	// There can be one lower version allowed if strict mode disabled
	var lower *semver.Version
	for _, r := range reports {
		if r.v == nil {
			g.Trace(fmt.Sprintf("  %s reports no version", r.n))
			continue
		}
		if version == nil {
			g.Log(fmt.Sprintf("  %s reports version: %s", r.n, r.v.String()))
			version = r.v
			continue
		}
		if version.Equal(r.v) {
			g.Log(fmt.Sprintf("  %s report version: %s", r.n, r.v.String()))
			continue
		}
		if !g.Strict && lower == nil && r.s.Source.IsCanBeLesser() &&
			r.v.LessThan(version) {
			lower = r.v
			g.Log(
				fmt.Sprintf(
					"  %s report lesser version: %s (allowed)",
					r.n,
					r.v.String(),
				),
			)
			continue
		}
		g.Err(
			fmt.Sprintf("  %s report different version: %s", r.n, r.v.String()),
		)
		if err == nil {
			err = errors.New("  sources reporting different versions")
		}
	}
	if vp && version != nil {
		version = addVPrefix(version)
	}
	if err != nil {
		version = nil
	}
	if version == nil {
		g.Log("  no version found in project, using default one")
		if g.DefaultVersion == "" {
			return semver.NewVersion("0.1.0")
		}
	}
	return version, nil
}

// Return only first error.
func (g *SourceGroup) Set(v semver.Version, names []Name) (err error) {
	g.Log("writing versions to sources...")
	sources := g.Sources
	if len(names) > 0 {
		sources = g.Filter(names)
	}
	for name, src := range sources {
		if src.Disabled {
			g.Trace(fmt.Sprintf("  %s skipped as disabled", name))
			continue
		}
		if src.Source.IsReadOnly() {
			g.Trace(fmt.Sprintf("  %s skipped as readonly", name))
			continue
		}
		var e error
		// Handle leading v
		if src.VPrefix == VPrefixAuto {
			e = src.Source.Set(v, g.setFS)
		}
		if src.VPrefix == VPrefixTrue {
			e = src.Source.Set(*addVPrefix(&v), g.setFS)
		}
		if src.VPrefix == VPrefixFalse {
			e = src.Source.Set(*trimVPrefix(&v), g.setFS)
		}
		if errors.Is(e, errNoChanges) {
			g.Trace(fmt.Sprintf("  %s: no changes", name))
			continue
		}
		if e != nil {
			g.Err(fmt.Sprintf("  %s failed with: %s", name, e))
			if err == nil {
				err = e
			}
			continue
		}
		g.Log(fmt.Sprintf("  %s: ok", name))
	}
	return
}

func (g *SourceGroup) verify() error {
	for name := range g.Sources {
		if !startsWithCapital(name) {
			return fmt.Errorf("%s must be in CamelCase", name)
		}
	}
	return nil
}
