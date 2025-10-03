# version

A tiny CLI to **read, compare, bump and set SemVer versions** across multiple
sources of truth (git tags, `package.json`, `pyproject.toml`, `Cargo.toml`,
arbitrary files, etc).
Useful in local workflows and CI to keep everything in sync.

## Features

* Read versions from many source types: JSON / TOML / YAML files, arbitrary text, external tools output, and git tags.
* Compare versions from multiple sources and report mismatches.
* `set` a new version across configured writable sources.
* `bump` a semantic component (major/minor/patch).
* `max` to compute the maximum version among literals and sources.
* Configurable defaults and per-source behavior (preserve `v` prefix, read-only files, ignored globs).

## Installation
### Binary
If none of the options listed below work for you, you can always just download
a statically linked executable for your platform from the [releases page](https://github.com/asciimoth/version/releases/latest).
### Nix
Nix users can install `version` with flake:
```sh
# Install to sys profile
nix profile add github:asciimoth/version
# Remove from sys profile
nix profile remove version

# Add to temporal shell
nix shell github:asciimoth/version
```
### Deb/Rpm
You can download deb/rmp packages from [releases page](https://github.com/asciimoth/version/releases/latest)
or use my [deb/rpm repo](https://repo.moth.contact/).
### Arch
Arch users can install [version-git](https://aur.archlinux.org/packages/version-git)
from AUR or download `.pkg` files from [releases page](https://github.com/asciimoth/version/releases/latest) too.  
### Go
You can also install it with go:
```sh 
go install github.com/asciimoth/version@latest
```

## Quick start / Usage
Basic usage (same as `get`):

```bash
version
```

Show global help:

```bash
version --help
```

Per-subcommand help:

```bash
version get --help
version set --help
version bump --help
version max --help
```

Examples:

```bash
# Read versions from defaults
version

# Check only package.json and git
version get PackageJson Git

# Set version to 2.0.0 across writable sources
version set 2.0.0

# Bump minor component using the maximum available version
version bump

# Choose maximum among arguments and sources
version max Git PackageJson 1.4.0
```

## Subcommands (summary)
- `get` - Read versions from sources and compare. Print agreed version or report mismatches. Accepts:
  - Source names to limit which sources to check.
  - `major | minor | patch` to print only a specific component.
  - A literal semver fallback (e.g. `1.2.3`) used when no source contains a value.
  - `--strict` toggles strict behavior (see below).
- `set <semver>` - Write the given semver into configured writable sources (or listed sources). Respects per-source `VPrefix` and read-only settings.
- `bump [major|minor|patch]` — Determine base version (maximum among sources or provided literal), increment chosen component (default `minor`), write result back to writable sources and print new version.
- `max [items...]` — Return the maximum among literals and listed sources. If no items, prints `DefaultVersion` (or `0.1.0`).

## Configuration
`version` can be configured either by a project-local `version.toml` file or by
placing a `tool.version` section inside `pyproject.toml`.

Top-level keys:
- `DefaultVersion` — string, semver fallback if no source reports a version. Default: `0.1.0`.
- `Strict` — bool, enable strict mode by default.
- `IgnoredFiles` — array of string globs to ignore (applies to all subcommands).
- `ReadOnlyFiles` — array of string globs; `set` and `bump` will not modify matching files.
- `Sources` — table mapping CamelCase source names to per-source config.

### Example `version.toml`
```toml
DefaultVersion = "1.1.1"
Strict = true
ReadOnlyFiles = ["fileA", "*.toml"]

[Sources.PackageJson]
Type = "json"
VPrefix = "false"
Path = "package.json"
KeyPath = ["version"]

[Sources.Git]
Type = "git"
VPrefix = "auto"
ReadOnly = false
```

### Example `pyproject.toml`
```toml
[tool.version]
DefaultVersion = "1.1.1"
Strict = true

[tool.version.Sources.PackageJson]
Type = "json"
Path = "package.json"
KeyPath = ["version"]
VPrefix = "false"
```

## Source types & per-source options
Each source config lives under `Sources.<Name>` where `<Name>` is a CamelCase
identifier (e.g. `PackageJson`, `PyProject`, `Cargo`, `Git`).

Common per-source fields:
- `Type` — one of: `json`, `toml`, `yaml`, `regexp`, `tool`, `git`.
- `VPrefix` — `true | false | auto`. Controls whether leading `v` is preserved/used when writing. (default `auto` tries to preserve existing style.)

Type-specific fields:
- `json`, `toml`, `yaml`:
  - `Path` — path to file.
  - `KeyPath` — array of keys to reach the value (for nested structures). Example: `["package","version"]`.
- `regexp`:
  - `Path` — path to file.
  - `KeyPath` — array of regular expressions that locate the substring to read/update.
- `tool`:
  - `Cmd` — array of strings: command and args to execute to obtain version.
  - `Pipe` — `stdout | stderr | both`. Default `stdout`.
  - `ExpectedStatus` — integer expected exit code.
  - `CD` — directory to `chdir` into before running.
  - `Env` — map of environment variables to set.
  - `Regexps` — array of regular expressions that locate the substring in command output.
- `git`:
  - `CD` — directory to run git in.
  - `Env` — env vars for git invocation.
  - `ReadOnly` — when true, `set` will not create tags.
  - Behavior: reads SemVer-compatible tags and, on `set`, creates a tag for the latest commit.

## Default sources
Used when no config file exists
```toml
[Sources.PackageJson]
Type = "json"
VPrefix = "false"
Path = "package.json"
KeyPath = ["version"]

[Sources.PyProject]
Type = "toml"
VPrefix = "false"
Path = "pyproject.toml"
KeyPath = ["project", "version"]

[Sources.Cargo]
Type = "toml"
VPrefix = "false"
Path = "Cargo.toml"
KeyPath = ["package", "version"]

[Sources.Git]
Type = "git"
VPrefix = "auto"
```

## VPrefix
`VPrefix` controls how a given source treats a leading `v` (e.g. `v1.2.3`):
- `true` — always write with a leading `v`.
- `false` — always write without `v`.
- `auto` — preserve whatever is already present.

## Strict mode
By default strict mode is **off**. This allows `version get` to be used as a
pre-commit linter when a git tag for the just-created commit is not available
yet (git tag would be written after commit).
In non-strict mode some sources may be lower than others without failing.

Use `--strict` (or set `Strict = true` in config) to require that **no**
source reports a version lower than the maximum present — useful for CI checks.

## License
This project is licensed under either of

- Apache License, Version 2.0, ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)

at your option.
