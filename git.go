package main

import (
	"bufio"
	"bytes"
	"os/exec"
	"regexp"

	"github.com/Masterminds/semver/v3"
)

var semverRegexp = regexp.MustCompile(
	`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`, //nolint:revive,lll
)

func init() {
	RegisterSource("git", func() Source { return &GitSource{} })
	RegisterDefaultSource("Git", SourceWithMeta{
		VPrefix: VPrefixAuto,
		Source:  &GitSource{},
	})
}

type GitSource struct {
	CD       string
	Env      map[string]string
	ReadOnly bool
}

func (g *GitSource) IsCanBeLesser() bool {
	return true
}

func (g *GitSource) IsReadOnly() bool {
	return g.ReadOnly
}

func (g *GitSource) Get(_ FS) (*semver.Version, error) {
	cmd, err := constructCmd([]string{"git", "tag"}, g.CD, g.Env)
	if err != nil {
		return nil, err
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	tags, err := parseSemverTagsFromReader(bytes.NewReader(out))
	if err != nil {
		return nil, err
	}
	var maxTag *semver.Version
	for _, tag := range tags {
		t, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		if maxTag == nil || t.GreaterThan(maxTag) {
			maxTag = t
			continue
		}
	}
	return maxTag, nil
}

func (g *GitSource) Set(v semver.Version, _ FS) error {
	if g.ReadOnly {
		return nil
	}
	str := verToString(&v)
	cmd := exec.Command("git", "tag", "-f", str) //nolint:gosec,noctx
	_, err := cmd.Output()
	return err
}

func parseSemverTagsFromReader(r *bytes.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		t := sc.Text()
		if semverRegexp.MatchString(t) {
			out = append(out, t)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
