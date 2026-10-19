<div align="center">

# ppm: Python Project Manager

**Create, run and manage Python projects with one small binary.**

[![Release](https://img.shields.io/github/v/release/rahulcodepython/PPM?sort=semver)](https://github.com/rahulcodepython/PPM/releases/latest)
[![Release build](https://github.com/rahulcodepython/PPM/actions/workflows/release.yml/badge.svg)](https://github.com/rahulcodepython/PPM/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

ppm gives Python the workflow `npm` gives JavaScript: one command to start a
project, one file (`ppm.yml`) that lists its dependencies and scripts, and
commands that always run inside the project's virtual environment. You never
have to type `source .venv/bin/activate` again.

```console
$ ppm init myapp -t fastapi -y
✓ Created ppm.yml, .env, .gitignore, README.md, src/main.py
› Creating virtual environment in .venv (Python 3.12.4, uv)
✓ Saved to ppm.yml dependencies
✓ Git repository initialized on branch main

$ cd myapp && ppm run
> python -m uvicorn main:app --app-dir src --reload
INFO:     Uvicorn running on http://127.0.0.1:8000
```

## Why ppm?

- **One binary, no setup.** Written in Go. It works with any Python 3 you have
  and installs nothing into your global Python.
- **Simple project file.** `ppm.yml` holds metadata, scripts, dependencies
  and dev dependencies, in plain YAML.
- **Always uses the venv.** `ppm run`, `ppm exec` and `ppm shell` activate
  `.venv` and load `.env` for you.
- **Fast when you want it.** ppm uses [uv](https://docs.astral.sh/uv/) when
  it is installed, and pip otherwise.
- **Works for teams.** After `git clone`, one `ppm install` rebuilds the same
  environment. `ppm freeze` writes an exact `requirements.txt`.
- **Runs everywhere.** Linux, macOS and Windows, on amd64 and arm64.

## Install

The installers download the right binary for your system, verify its
checksum, put it in the standard location and **add it to your PATH**. Open a
new terminal afterwards and `ppm` works from any folder.

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.sh | sh
```

**Windows (PowerShell, no admin needed)**

```powershell
irm https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.ps1 | iex
```

| System | Installed to | PATH is set in |
| --- | --- | --- |
| Linux | `~/.local/bin/ppm` | `~/.profile`, `~/.bashrc`, `~/.zshrc`, fish `conf.d` |
| macOS | `~/.local/bin/ppm` | `~/.profile`, `~/.zshrc` (and bash/fish if used) |
| Windows | `%LOCALAPPDATA%\Programs\ppm\ppm.exe` | your user `Path` environment variable |
| Linux/macOS as root | `/usr/local/bin/ppm` | already on PATH |

Then check it with:

```bash
ppm --version
ppm doctor
```

**Other ways to install**

| Method | How |
| --- | --- |
| Specific version | `curl -fsSL .../install.sh \| PPM_VERSION=v2.0.0 sh`, or on Windows `$env:PPM_VERSION="v2.0.0"` before the `irm` line |
| Custom folder | `PPM_INSTALL_DIR=/opt/bin` (works with both scripts) |
| Debian / Ubuntu | download `ppm_<version>_amd64.deb` from [Releases](https://github.com/rahulcodepython/PPM/releases/latest), `sudo apt install ./ppm_*.deb` |
| Fedora / RHEL | `sudo dnf install ./ppm_*.rpm` |
| Alpine | `sudo apk add --allow-untrusted ./ppm_*.apk` |
| Manual download | get the `.tar.gz` / `.zip` from [Releases](https://github.com/rahulcodepython/PPM/releases/latest), extract, run `./ppm self install` (Windows: `.\ppm.exe self install`) |
| From source | see [Build from source](#build-from-source) |

**Update:** `ppm self update`. **Uninstall:** `ppm self uninstall`. This
removes the binary and the PATH entry. Your projects are not touched.

**Requirements:** Python 3.9 or newer on your `PATH`. git and uv are optional.

## Quick start

```bash
ppm init myapp            # answer a few questions (or add -y)
cd myapp
ppm add requests          # install + save to ppm.yml
ppm add -D pytest         # dev-only dependency
ppm script add test python -m pytest
ppm run                   # run the "start" script
ppm run test              # run another script
```

Working on an existing project:

```bash
git clone https://github.com/you/project && cd project
ppm install               # creates .venv and installs everything
```

Already have a `requirements.txt`? Run `ppm init` in that folder. ppm
imports it.

## Commands

| Command | What it does |
| --- | --- |
| **Project** | |
| `ppm init [dir]` | Create a project. `-t app\|cli\|fastapi\|flask`, `-y`, `-p 3.12`, `--no-git` |
| `ppm use <python>` | Switch Python version and rebuild `.venv` |
| `ppm env list\|set\|unset` | Manage variables in `.env` |
| `ppm clean` | Delete caches and build output (`--venv` also deletes `.venv`) |
| `ppm doctor` | Check Python, uv, git and the project |
| **Dependencies** | |
| `ppm add <pkg>...` | Install and save packages (`-D` for dev) |
| `ppm install` | Install everything in `ppm.yml` (`--prod`, `--clean`, `-r file`) |
| `ppm remove <pkg>...` | Uninstall and drop from `ppm.yml` |
| `ppm update [pkg]...` | Upgrade dependencies |
| `ppm list` | Table of dependencies: required spec, installed version, type (`--all` adds sub-dependencies, `--json`) |
| `ppm outdated` | Table of packages with newer versions (`--json`) |
| `ppm freeze` | Write pinned `requirements.txt` |
| **Running** | |
| `ppm run [script] [args]` | Run a script (default `start`) or a `.py` file |
| `ppm exec <cmd>` | Run any command in the venv, e.g. `ppm exec pytest -x` |
| `ppm shell` | Open a shell with the venv activated |
| `ppm script list\|add\|remove` | Manage scripts in `ppm.yml` |
| **ppm itself** | |
| `ppm self install\|update\|uninstall` | Install this binary to PATH, update it, or remove it |
| `ppm completion <shell>` | Shell completion for bash, zsh, fish, powershell |

Every command has built-in help with examples: `ppm help <command>` or
`ppm <command> --help`. There is also `man ppm` when installed from a package.

## ppm.yml

```yaml
name: myapp
version: 0.1.0
description: My web API
author: Rahul Das
license: MIT
python: "3.12"           # optional: version or command used for .venv
installer: auto          # optional: auto (uv if found), pip or uv
venv: .venv
env_file: .env
scripts:
  start: python -m uvicorn main:app --app-dir src --reload
  test: python -m pytest
  pretest: ruff check .  # runs before "test"
dependencies:
  - fastapi>=0.115.0
  - uvicorn[standard]>=0.30.6
dev-dependencies:
  - pytest>=8.3.2
  - ruff>=0.6.4
```

See the [user manual](docs/USAGE.md) for every field, script hooks, `.env`
rules and recipes.

## Coming from ppm3 (the Python version)?

Install the binary, then run any ppm command in your project. The old
`ppm.yml` is converted automatically. Your `run` command becomes the `start`
script. Then run `pip uninstall ppm3`. The old format recorded every
installed package, so trim `dependencies` to the packages you import.
Details: [Migrating from ppm3](docs/USAGE.md#migrating-from-ppm3).

## Build from source

You need [Go](https://go.dev/dl/) 1.23+ and git.

### Install from source (anyone)

Linux / macOS:

```bash
git clone https://github.com/rahulcodepython/PPM.git
cd PPM
make install        # builds ./ppm, copies it to ~/.local/bin, adds it to PATH
```

Windows (PowerShell), or any system without `make`:

```powershell
git clone https://github.com/rahulcodepython/PPM.git
cd PPM
go build -o ppm.exe ./cmd/ppm      # on Linux/macOS: go build -o ppm ./cmd/ppm
.\ppm.exe self install             # on Linux/macOS: ./ppm self install
```

Open a new terminal and run `ppm --version`. `self install` puts the binary
in the same place as the installers, so `ppm self uninstall` works too.

Or with Go only: `go install github.com/rahulcodepython/PPM/cmd/ppm@latest`.
This puts `ppm` in `$(go env GOPATH)/bin`, which must be on your PATH.

### Local development mode

```bash
go run ./cmd/ppm --help     # run your working copy without installing
go run ./cmd/ppm -C /path/to/a/python/project list

make dev                    # rebuild and reinstall as `ppm`; rerun after each change
make test                   # go test ./...
make lint                   # gofmt + go vet
make snapshot               # full release build into ./dist (needs goreleaser)
```

On Windows without make, `make dev` is the same as running the
`go build` and `self install` lines above again. `make` builds report a
plain version number: the release tag when you build a tagged commit,
otherwise the version the next release will get (for example `ppm 2.0.0`).
`ppm version` also shows the git commit. A plain `go build` without `make`
reports `dev`.

## Releases

Every push to `main` that changes code is released automatically. GitHub
Actions runs the tests on Linux, macOS and Windows, picks the next version
from the commit messages (`fix:` = patch, `feat:` = minor, `feat!:` = major),
tags it, and publishes the binaries to
[Releases](https://github.com/rahulcodepython/PPM/releases). Details are in
[docs/PUBLISHING.md](docs/PUBLISHING.md).

## Contributing

Issues and pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).
Please follow the [Code of Conduct](CODE_OF_CONDUCT.md), and report security
issues privately as described in [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © 2024-2026 Rahul Das. Third-party licenses are listed in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
