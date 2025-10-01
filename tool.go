package main

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/asciimoth/inplace/regexp"
)

func init() {
	RegisterSource("tool", func() Source { return &ToolSource{} })
}

func constructCmd(
	args []string,
	pwd string,
	env map[string]string,
) (cmd *exec.Cmd, err error) {
	// prepare command
	if len(args) < 2 {
		cmd = exec.Command(args[0]) //nolint:gosec,noctx
	} else {
		cmd = exec.Command(args[0], args[1:]...) //nolint:gosec,noctx
	}

	// working dir: treat CD as relative to current working directory unless it's absolute
	if pwd != "" {
		if filepath.IsAbs(pwd) {
			cmd.Dir = pwd
		} else {
			wd, gerr := os.Getwd()
			if gerr != nil {
				err = fmt.Errorf("getting current working directory: %w", gerr)
				return
			}
			cmd.Dir = filepath.Join(wd, pwd)
		}
	}

	// merge environment: start from current env, override with t.Env
	envMap := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		k := parts[0]
		v := ""
		if len(parts) > 1 {
			v = parts[1]
		}
		envMap[k] = v
	}
	maps.Copy(envMap, env)
	envList := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envList = append(envList, k+"="+v)
	}
	cmd.Env = envList

	return
}

type ToolSource struct {
	Cmd            []string
	Pipe           string
	ExpectedStatus int
	Regexps        []string
	CD             string
	Env            map[string]string
}

func (d *ToolSource) IsCanBeLesser() bool {
	return false
}

func (d *ToolSource) IsReadOnly() bool {
	return true
}

func (d *ToolSource) Set(_ semver.Version, _ FS) error {
	return nil // Read Only
}

func (d *ToolSource) Get(_ FS) (*semver.Version, error) {
	out, err := d.exec()
	if err != nil {
		return nil, err
	}
	if len(d.Regexps) == 0 {
		return semver.NewVersion(strings.TrimSpace(string(out)))
	}
	doc, err := regexp.New(out)
	if err != nil {
		return nil, err
	}
	str := doc.Get(d.Regexps)
	return semver.NewVersion(str)
}

func (d *ToolSource) exec() ([]byte, error) {
	if len(d.Cmd) == 0 {
		return nil, errors.New("no command specified")
	}
	// prepare command
	cmd, err := constructCmd(d.Cmd, d.CD, d.Env)
	if err != nil {
		return nil, err
	}

	// capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// run
	runErr := cmd.Run()

	// collect outputs regardless of runErr (we still want captured output)
	stdoutBytes := stdoutBuf.Bytes()
	stderrBytes := stderrBuf.Bytes()

	// determine exit code
	var exitCode int
	if runErr != nil {
		// If the process started but exited with non-zero, get exit code when possible
		ee := &exec.ExitError{}
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitCode()
		} else {
			// other error (e.g. executable not found, permission)
			// return the captured output and the error
			return choosePipe(d.Pipe, stdoutBytes, stderrBytes), fmt.Errorf("failed to run command: %w", runErr)
		}
	} else {
		// success -> exitCode stays 0
		exitCode = 0
	}

	// compare exit code with ExpectedStatus
	if exitCode != d.ExpectedStatus {
		// include some helpful context in the error
		return choosePipe(d.Pipe, stdoutBytes, stderrBytes),
			fmt.Errorf(
				"unexpected exit status: got %d, expected %d",
				exitCode,
				d.ExpectedStatus,
			)
	}

	return choosePipe(d.Pipe, stdoutBytes, stderrBytes), nil
}

func choosePipe(pipe string, stdout, stderr []byte) []byte {
	if pipe == "" || pipe == "stdout" {
		return stdout
	}
	switch pipe {
	case "stderr":
		return stderr
	case "both":
		var b bytes.Buffer
		b.Write(stdout)
		b.Write(stderr)
		return b.Bytes()
	}
	return stdout
}
