package main

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/inplace"
	"github.com/asciimoth/rewrite"
)

var (
	errNoChanges = errors.New("no changes")
	errUnsync    = errors.New("multiple files reports different versions")
)

// true if the first rune is a letter and is uppercase.
func startsWithCapital(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLetter(r) && unicode.IsUpper(r)
}

func hasVPrefix(v *semver.Version) bool {
	if v == nil {
		return false
	}
	return strings.HasPrefix(v.Original(), "v")
}

func addVPrefix(v *semver.Version) *semver.Version {
	if hasVPrefix(v) {
		return v
	}
	return semver.MustParse("v" + v.String())
}

func trimVPrefix(v *semver.Version) *semver.Version {
	if !hasVPrefix(v) {
		return v
	}
	return semver.MustParse(strings.TrimLeft(v.String(), "v"))
}

func verToString(version *semver.Version) string {
	if hasVPrefix(version) {
		return "v" + version.String()
	}
	return version.String()
}

func getFromDoc(
	fs FS,
	con inplace.New,
	kp inplace.KeyPath,
	path string,
) (*semver.Version, error) {
	files, err := fs.Glob(path)
	if err != nil {
		return nil, err
	}
	var v *semver.Version
	for _, path := range files {
		bytes, err := rewrite.Read(fs, path)
		if err != nil {
			if len(files) < 2 {
				return nil, err
			}
			continue
		}
		doc, err := con(bytes)
		if err != nil {
			return nil, err
		}
		val := doc.Get(kp)
		if val != "" {
			cv, err := semver.NewVersion(val)
			if err != nil {
				return nil, err
			}
			if v == nil {
				v = cv
				continue
			}
			if !cv.Equal(v) {
				return nil, errUnsync
			}
		}
	}
	return v, nil
}

func setToDoc(
	v semver.Version,
	fs FS,
	con inplace.New,
	kp inplace.KeyPath,
	path string,
) error {
	files, err := fs.Glob(path)
	if err != nil {
		return err
	}
	val := verToString(&v)
	changes := false
	for _, path := range files {
		bytes, err := rewrite.Read(fs, path)
		if err != nil {
			continue
		}
		doc, err := con(bytes)
		if err != nil {
			continue
		}
		prev := doc.Get(kp)
		if prev == "" {
			continue
		}
		err = doc.Set(kp, val)
		if err != nil {
			return err
		}
		bytes = doc.Save()
		err = rewrite.Write(fs, path, bytes)
		if err != nil {
			return err
		}
		changes = true
	}
	if changes {
		return nil
	}
	return errNoChanges
}
