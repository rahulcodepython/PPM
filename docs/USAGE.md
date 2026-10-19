# ppm user manual

This manual covers everything ppm does. For one command's flags, run
`ppm <command> --help`.

- [Installation](#installation)
- [Concepts](#concepts)
- [Creating a project](#creating-a-project)
- [Managing dependencies](#managing-dependencies)
- [Running code](#running-code)
- [Environment variables](#environment-variables)
- [Python versions and installers](#python-versions-and-installers)
- [ppm.yml reference](#ppmyml-reference)
- [Recipes](#recipes)
- [Migrating from ppm3](#migrating-from-ppm3)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Install script (recommended)

Linux and macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.sh | sh
```

Windows (PowerShell, no admin needed):

```powershell
irm https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.ps1 | iex
```

The script detects your OS and CPU, downloads the matching release and checks
its SHA-256 checksum. It then runs `ppm self install`, which copies the binary
to the standard location and adds it to PATH:

| System | Binary | How PATH is set |
| --- | --- | --- |
| Linux, macOS | `~/.local/bin/ppm` | a marked block in `~/.profile`, plus `~/.bashrc`, `~/.bash_profile`, `~/.zshrc` when present or when they belong to your shell (zsh on macOS), and `~/.config/fish/conf.d/ppm.fish` for fish |
| Linux, macOS as root | `/usr/local/bin/ppm` | already on PATH |
| Windows | `%LOCALAPPDATA%\Programs\ppm\ppm.exe` | appended to your **user** `Path` variable. The current PowerShell window is updated too, and other open apps are notified |

Open a new terminal, then check with `ppm --version`. Running the installer
again is safe: it replaces the binary and does not add PATH twice.

Options, as environment variables:

```bash
# a specific version into a custom folder
curl -fsSL https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.sh \
  | PPM_VERSION=v2.0.0 PPM_INSTALL_DIR=/opt/bin sh
```

```powershell
$env:PPM_VERSION = "v2.0.0"; $env:PPM_INSTALL_DIR = "C:\tools\ppm"
irm https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.ps1 | iex
```

> Piping a script into a shell runs it with your permissions. To read it
> first, download `install.sh` / `install.ps1`, review it, then run it.

### Linux packages

Every release has `.deb`, `.rpm` and `.apk` files. They install to
`/usr/bin/ppm` and include the man pages and shell completions:

```bash
sudo apt install ./ppm_2.0.0_amd64.deb                 # Debian, Ubuntu
sudo dnf install ./ppm_2.0.0_amd64.rpm                 # Fedora, RHEL
sudo apk add --allow-untrusted ./ppm_2.0.0_amd64.apk   # Alpine
```

### Manual download

1. Open the [latest release](https://github.com/rahulcodepython/PPM/releases/latest).
2. Download `ppm_<os>_<arch>.tar.gz` (or `.zip` on Windows). Use `amd64` for
   Intel/AMD and `arm64` for Apple Silicon, Raspberry Pi 4+ or Windows on ARM.
3. Optional: compare its SHA-256 with `checksums.txt`.
4. Extract it and run `./ppm self install` (Windows: `.\ppm.exe self install`).
   Then you can delete the downloaded file.

`ppm self install` options: `--dir <folder>` to choose the folder,
`--no-path` to leave PATH alone.

On macOS, a binary from a browser download can be blocked by Gatekeeper.
Run `xattr -d com.apple.quarantine ./ppm` once, or use the install script,
which isn't affected.

### From source

See [Build from source](../README.md#build-from-source): `make install`, or
`go build -o ppm ./cmd/ppm && ./ppm self install`.

### Shell completion

Packages set this up for you. For other installs:

| Shell | Command |
| --- | --- |
| bash | `ppm completion bash > ~/.local/share/bash-completion/completions/ppm` |
| zsh | `ppm completion zsh > "${fpath[1]}/_ppm"` (then restart the shell) |
| fish | `ppm completion fish > ~/.config/fish/completions/ppm.fish` |
| PowerShell | `ppm completion powershell \| Out-String \| Invoke-Expression` (add to `$PROFILE`) |

Completion covers commands, flags, template names and the script names in
your `ppm.yml`.

### Updating and uninstalling

- Update: `ppm self update` (or `ppm self update --check` to only look).
  If you installed from a package, update with the package manager instead.
- Uninstall: `ppm self uninstall` removes the binary and the PATH entry
  (on Windows, delete the leftover `%LOCALAPPDATA%\Programs\ppm` folder
  afterwards). For packages: `sudo apt remove ppm` / `dnf remove` / `apk del`.
  ppm keeps no global state, and each project's `.venv` is a normal folder.

---

## Concepts

| Thing | Meaning |
| --- | --- |
| **project** | a folder with a `ppm.yml`. ppm finds it from any subfolder, like git does |
| **ppm.yml** | metadata, scripts, direct dependencies and dev dependencies |
| **.venv** | the project's virtual environment. ppm creates it when needed |
| **.env** | `KEY=VALUE` variables loaded when ppm runs your code |
| **requirements.txt** | exact pinned versions, written by `ppm freeze` |

ppm creates `.venv` the first time a command needs it, and installs the
dependencies when it does. So after a fresh clone, `ppm run` just works.

Global flag: `-C <dir>` / `--cwd <dir>` runs any command as if started in
`<dir>`, e.g. `ppm -C services/api run`.

---

## Creating a project

```bash
ppm init                 # in the current folder, interactive
ppm init myapp           # create ./myapp
ppm init myapp -y        # accept every default
```

ppm asks for the name, version, description, author (taken from
`git config user.name`), license, Python version, and whether to create a git
repo with an optional remote URL. Press Enter to keep the default shown in
brackets.

It creates:

```
myapp/
├── .env            # empty, git-ignored
├── .gitignore      # Python, venv, editor files
├── .venv/          # virtual environment
├── README.md
├── ppm.yml
└── src/main.py     # depends on the template
```

It never overwrites existing files. Files that already exist are kept and
listed. If the folder has a `requirements.txt`, ppm offers to import it into
`ppm.yml`. With `-y`, it imports it without asking.

### Templates

| `-t` | Creates | Dependencies | `start` script |
| --- | --- | --- | --- |
| `app` (default) | `src/main.py` | none | `python src/main.py` |
| `cli` | argparse tool in `src/main.py` | none | `python src/main.py` |
| `fastapi` | API in `src/main.py` | fastapi, uvicorn[standard] | `python -m uvicorn main:app --app-dir src --reload` |
| `flask` | app in `src/app.py` | flask | `python -m flask --app src/app.py run --debug` |

### Other init flags

| Flag | Effect |
| --- | --- |
| `-p, --python 3.12` | Python version or command for the venv |
| `--installer pip\|uv\|auto` | package installer saved in `ppm.yml` |
| `--no-git` | skip `git init` |
| `--remote <url>` | add a git `origin` |
| `--no-venv` | write files only, create `.venv` later with `ppm install` |

With git, ppm creates branch `main` and makes an initial commit. If your git
has no `user.name`/`user.email`, the repo is created without a commit and ppm
shows a warning.

---

## Managing dependencies

### Adding

```bash
ppm add requests                      # saved as requests>=<installed version>
ppm add "django>=5,<6"                # your spec is saved as written
ppm add "uvicorn[standard]"           # extras
ppm add git+https://github.com/org/lib.git
ppm add ./libs/shared                 # local path
ppm add -D pytest ruff mypy           # dev dependencies
```

Adding a package that is already listed replaces its entry, even if it is in
the other list. `ppm install <pkg>` is the same as `ppm add <pkg>`.

### Installing a project's dependencies

```bash
ppm install              # dependencies + dev-dependencies
ppm install --prod       # skip dev-dependencies (servers, Docker images)
ppm install --clean      # delete .venv and rebuild from scratch
ppm install -r requirements.txt   # import a file into ppm.yml and install
```

### Removing, updating, inspecting

```bash
ppm remove requests            # alias: ppm rm, ppm uninstall
ppm update                     # upgrade all dependencies
ppm update fastapi             # upgrade one
ppm list                       # table: package, required spec, installed version, type
ppm list --all                 # also every sub-dependency in .venv
ppm outdated                   # table with the latest version of each outdated package
ppm list --json                # same data as JSON (also: ppm outdated --json)
```

```text
┌───────────────────┬──────────┬───────────┬──────┐
│ PACKAGE           │ REQUIRED │ INSTALLED │ TYPE │
├───────────────────┼──────────┼───────────┼──────┤
│ requests          │ >=2.32.3 │ 2.32.3    │ prod │
│ uvicorn[standard] │ >=0.30.6 │ 0.30.6    │ prod │
│ pytest            │ >=8.3.2  │ 8.3.2     │ dev  │
└───────────────────┴──────────┴───────────┴──────┘
3 packages: 2 prod, 1 dev
```

`TYPE` is `prod` (in `dependencies`), `dev` (in `dev-dependencies`) or `sub`
(not in `ppm.yml`; installed because another package needs it). A package
listed in `ppm.yml` but absent from `.venv` shows as `missing`.

`ppm update` keeps your specs, but moves simple ones forward:
`requests>=2.31.0` becomes `requests>=2.32.3`. Ranges such as `>=2,<3` stay
as they are.

`ppm remove` uninstalls only the package you name. Its sub-dependencies stay
in `.venv`. To drop those too, run `ppm install --clean`.

### Reproducible installs

`ppm.yml` lists what you want, and `requirements.txt` records exactly what you
got:

```bash
ppm freeze                      # write requirements.txt
ppm freeze -o requirements.lock
ppm freeze -o -                 # print instead
pip install -r requirements.txt # deploy target without ppm
```

---

## Running code

### Scripts

Scripts live under `scripts:` in `ppm.yml`:

```bash
ppm run                  # runs "start"
ppm run test             # runs "test"
ppm run test -k login    # extra args are passed to the script
ppm run -- --port 9000   # extra args to "start"
ppm run tools/seed.py    # any .py file, with the venv's Python
ppm run -l               # list scripts
```

Manage them without editing YAML:

```bash
ppm script add test python -m pytest -q
ppm script add check "ruff check . && ppm run test"   # quote shell operators
ppm script remove check
ppm script list
```

Scripts run through the system shell (`/bin/sh` on Linux/macOS, `cmd.exe` on
Windows), so `&&`, `|` and `$VAR` work. On Windows use `%VAR%`.

**Hooks:** if `pre<name>` or `post<name>` scripts exist, they run before or
after `<name>`. If any step fails, ppm stops and returns that exit code.

```yaml
scripts:
  prestart: python -m alembic upgrade head
  start: python -m uvicorn main:app --app-dir src
  test: python -m pytest
  posttest: python -m coverage report
```

Exit codes pass through, so `ppm run test` works in CI. Ctrl+C goes to your
program, not to ppm.

### Any command

```bash
ppm exec pytest -x
ppm exec python -m pip show requests
ppm exec black --check .        # alias: ppm x
```

Programs installed in `.venv` are found before system ones.

### A shell inside the venv

```bash
ppm shell        # starts $SHELL (or cmd.exe) with .venv active and .env loaded
exit             # leave it
```

---

## Environment variables

`ppm run`, `ppm exec` and `ppm shell` read `.env` (or the `env_file` set in
`ppm.yml`). Variables already set in your environment win, so
`DEBUG=0 ppm run` overrides the file.

```bash
ppm env set DEBUG=1 "GREETING=hello world"
ppm env list           # names only; values hidden
ppm env list --show    # with values
ppm env unset DEBUG
```

Supported `.env` syntax:

```bash
# comments and blank lines
PLAIN=value
export ALSO_OK=value            # "export" prefix is ignored
SPACED="double quotes, \n escapes"
LITERAL='single quotes, no $expansion'
INLINE=value # trailing comments on unquoted values
```

`.env` is git-ignored by default. Keep secrets in it, never in `ppm.yml`.
Commit a `.env.example` with the variable names for your team.

---

## Python versions and installers

### Choosing Python

ppm uses `python3` (`py -3` on Windows) unless you set one:

```bash
ppm init -p 3.12          # when creating
ppm use 3.11              # later: saves to ppm.yml, rebuilds .venv, reinstalls
ppm use /opt/py/bin/python3
```

A version like `3.12` looks for `python3.12` (`py -3.12` on Windows). You
can also give a command name or a full path. To install several Python
versions, use your OS packages, [python.org](https://www.python.org/downloads/),
`pyenv`, or `uv python install 3.12`.

### pip or uv

[uv](https://docs.astral.sh/uv/) installs packages much faster than pip.
With `installer: auto` (the default), ppm uses uv when it is on your `PATH`,
and pip otherwise. `ppm doctor` shows which one is active.

| Setting | Effect |
| --- | --- |
| `installer: auto` or unset | uv if installed, else pip |
| `installer: pip` | always pip |
| `installer: uv` | always uv; error if it is missing |
| `PPM_INSTALLER=pip ppm install` | one-off override |

A venv made with uv still includes pip, so `ppm exec pip ...` keeps working.

---

## ppm.yml reference

| Field | Default | Meaning |
| --- | --- | --- |
| `name` | folder name | project name |
| `version` | `0.1.0` | your project's version |
| `description` | empty | one-line summary |
| `author` | git user.name | author |
| `license` | `MIT` | license identifier (SPDX, e.g. `Apache-2.0`) |
| `python` | unset | Python version/command used to create `.venv` |
| `installer` | `auto` | `auto`, `pip` or `uv` |
| `venv` | `.venv` | virtual environment folder, relative to the project |
| `env_file` | `.env` | dotenv file loaded when running code |
| `scripts` | `start` | name to shell command |
| `dependencies` | `[]` | PEP 508 requirements needed to run |
| `dev-dependencies` | `[]` | requirements for development only |

ppm rewrites `ppm.yml` when you change dependencies or scripts. Lists are
sorted, and comments you add by hand are not preserved.

---

## Recipes

**Run tests in CI (GitHub Actions)**

```yaml
- uses: actions/setup-python@v5
  with: { python-version: "3.12" }
- run: curl -fsSL https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.sh | sh
- run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"
- run: ppm install
- run: ppm run test
```

**Docker**

`ppm freeze` pins everything in `.venv`, dev tools included. For a
production-only file, run `ppm install --clean --prod && ppm freeze` first.

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt   # from `ppm freeze`
COPY src/ src/
CMD ["python", "src/main.py"]
```

**Monorepo:** each service has its own `ppm.yml`. Run them from the root
with `ppm -C services/api run` and `ppm -C services/worker run`.

**Lint + format scripts**

```bash
ppm add -D ruff
ppm script add lint ruff check .
ppm script add fmt ruff format .
```

---

## Migrating from ppm3

ppm 2.x replaces the `ppm3` PyPI package.

1. Install ppm 2 (see [Installation](#installation)) and `pip uninstall ppm3`.
2. In your project, run any command, e.g. `ppm install`. The old `ppm.yml`
   is read, and saved in the new format on the next save.
3. Clean up `dependencies`. ppm3 saved every installed package, so delete
   the sub-dependencies you don't import directly.

| ppm3 | ppm 2 |
| --- | --- |
| `project.name`, `version`, ... | top-level `name`, `version`, ... |
| `commands.run` (`ppm run`) | `scripts.start` (`ppm run`) |
| `commands.<x>` | `scripts.<x>` |
| `packages` | `dependencies` |
| `environment.environment_name` | `venv` |
| `ppm init -y` | `ppm init -y` (unchanged) |
| `ppm install` / `uninstall` / `update` / `list` / `freeze` / `outdated` | same names |

---

## Troubleshooting

Run `ppm doctor` first. It shows which Python, installer, git and venv ppm
sees.

| Problem | Fix |
| --- | --- |
| `python 3 not found on PATH` | install Python 3, or point ppm at it: `ppm use /path/to/python3` |
| `ensurepip is not available` (Debian/Ubuntu) | `sudo apt install python3-venv` |
| `no ppm.yml found` | run inside the project, use `ppm -C <dir>`, or `ppm init` |
| Broken or moved `.venv` | `ppm install --clean` |
| Windows opens the Microsoft Store for `python` | install Python from python.org, or disable the "App execution aliases" for python |
| Want pip even though uv is installed | set `installer: pip` in `ppm.yml` |
| `self update` permission denied | re-run with `sudo`, or reinstall with the install script |
| `ppm` not found after installing | open a new terminal; check `ppm self install` printed "Added ... to PATH" |

Still stuck? [Open an issue](https://github.com/rahulcodepython/PPM/issues)
with the output of `ppm doctor`.
