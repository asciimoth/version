// Multi-source semantic version management tool
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/colorit"
)

// Mapping CLI command name -> it's implementation.
var commands = map[string]func(
	SourceGroup, []string, []string, []semver.Version, io.Writer,
) (int, error){
	"get":  cmdGet,
	"set":  cmdSet,
	"bump": cmdBump,
	"max":  cmdMax,
}

// List of SemVer version parts.
var elements = []string{"major", "minor", "patch"}

// CLI help messages.
var (
	//go:embed helps/main.txt
	helpMain string
	//go:embed helps/get.txt
	helpGet string
	//go:embed helps/set.txt
	helpSet string
	//go:embed helps/bump.txt
	helpBump string
	//go:embed helps/max.txt
	helpMax string
)

// Function to parse CLI args:
// - `cmd` - name of subcomamnd
// - `help` - was `--help` or `-h` flags provided
// - `strict` - was `--strict` or `-s` flags provided
// - `elements` - list of uinique provided SemVer parts names
// - `srcs` - list of names of sources
// - `vs` - list of provided SemVer version constants e.g. {"1.2.3", "6.5.4"}
// - `err` - error.
func parseCmd(args []string) (
	cmd string,
	help bool,
	strict bool,
	elems []string,
	srcs []string,
	vs []semver.Version,
	err error,
) {
	srcs = []string{}
	elems = []string{}
	vs = []semver.Version{}
	if len(args) > 0 {
		if slices.Contains(slices.Collect(maps.Keys(commands)), args[0]) {
			cmd = args[0]
			args = args[1:]
		}
	}
	for _, arg := range args {
		// Normalised arg
		narg := strings.ToLower(strings.TrimLeft(strings.TrimSpace(arg), "-"))
		if narg == "help" || narg == "h" {
			help = true
			return
		}
		if narg == "strict" || narg == "s" {
			strict = true
			continue
		}
		if slices.Contains(elements, narg) && !slices.Contains(elems, narg) {
			elems = append(elems, narg)
			continue
		}
		if startsWithCapital(arg) || arg == "none" {
			srcs = append(srcs, arg)
			continue
		}
		v, e := semver.NewVersion(narg)
		if e == nil {
			vs = append(vs, *v)
			continue
		}
		err = fmt.Errorf("unknown arg type \"%s\"", arg)
		return
	}
	if cmd == "" {
		cmd = "get"
	}
	return
}

// `--help` flag handler.
func showHelp(cmd string, out io.Writer) error {
	text := helpMain
	switch cmd {
	case "get":
		text = helpGet
	case "set":
		text = helpSet
	case "bump":
		text = helpBump
	case "max":
		text = helpMax
	}
	return colorit.HighlightTo(text, "help", out)
}

// `get` subcomamnd handler.
func cmdGet(
	group SourceGroup,
	elems []string,
	srcs []string,
	ver []semver.Version,
	out io.Writer,
) (int, error) {
	if len(ver) > 1 {
		return 1, errors.New(
			"this command can accept only zero or one version arg",
		)
	}
	if len(ver) == 1 {
		group.DefaultVersion = ver[0].Original()
	}
	vers, err := group.Get(srcs)
	if err != nil {
		return 1, err
	}
	if len(elems) < 1 {
		_, err := fmt.Fprintln(out, verToString(vers))
		if err != nil {
			return 1, err
		}
		return 0, nil
	}
	for _, elem := range elems {
		switch elem {
		case "major":
			group.Log(strconv.FormatUint(vers.Major(), 10))
		case "minor":
			group.Log(strconv.FormatUint(vers.Minor(), 10))
		case "patch":
			group.Log(strconv.FormatUint(vers.Patch(), 10))
		}
	}
	return 0, nil
}

// `bump` subcomamnd handler.
func cmdBump(
	group SourceGroup,
	elems []string,
	srcs []string,
	ver []semver.Version,
	out io.Writer,
) (int, error) {
	if len(ver) > 1 {
		return 1, errors.New(
			"this command can accept only zero or one version arg",
		)
	}
	if len(ver) == 1 {
		group.DefaultVersion = ver[0].Original()
	}
	if len(elems) < 1 {
		elems = []string{"minor"}
	}
	vers, err := group.Get(srcs)
	if err != nil {
		return 1, err
	}
	for _, elem := range elems {
		nv := *vers
		switch elem {
		case "major":
			nv = vers.IncMajor()
		case "minor":
			nv = vers.IncMinor()
		case "patch":
			nv = vers.IncPatch()
		}
		vers = &nv
	}
	err = group.Set(*vers, srcs)
	if err != nil {
		return 1, err
	}
	_, err = fmt.Fprintln(out, verToString(vers))
	if err != nil {
		return 1, err
	}
	return 0, nil
}

// `set` subcomamnd handler.
func cmdSet(
	group SourceGroup,
	_ []string,
	srcs []string,
	ver []semver.Version,
	_ io.Writer,
) (int, error) {
	if len(ver) != 1 {
		return 1, errors.New("this command accepts single version arg")
	}
	err := group.Set(ver[0], srcs)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

// `max` subcomamnd handler.
func cmdMax(
	group SourceGroup,
	_ []string,
	srcs []string,
	ver []semver.Version,
	_ io.Writer,
) (int, error) {
	v, err := group.GetMax(srcs, ver)
	if err != nil {
		return 1, err
	}
	group.Log(verToString(v))
	return 0, nil
}

// Function that select and call sultable subcommand handler.
func routeCmd(args []string, fs FS, sout, serr io.Writer) (int, error) {
	log := func(s string) { _, _ = fmt.Fprintln(serr, s) }
	cmd, help, strict, elems, srcs, vs, err := parseCmd(args)
	if err != nil {
		return 1, err
	}
	if help {
		err := showHelp(cmd, sout)
		if err != nil {
			return 1, err
		}
		return 0, nil
	}
	group, err := GroupFromConfig(fs, log, log, log, strict)
	if err != nil {
		return 1, err
	}
	f, ok := commands[cmd]
	if ok {
		return f(*group, elems, srcs, vs, sout)
	}
	return 1, fmt.Errorf("unknown subcommand %s", cmd)
}

func main() {
	root, err := os.OpenRoot(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	code, err := routeCmd(
		os.Args[1:],
		FSFromRoot(root),
		os.Stdout,
		os.Stderr,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}
