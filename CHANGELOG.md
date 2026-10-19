# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- `ppm list` and `ppm outdated` print aligned tables (colors no longer break
  the column widths), with `--json` output. `list --all` labels
  sub-dependencies. `script list` and `env list` use the same table.
- Commands that need arguments say what is missing and show usage and an
  example, e.g. `ppm uninstall` with no package. Their help has an
  `Arguments:` section.
- `ppm --help` lists subcommands (`env set`, `script add`, `self install`, ...)
  under their parent command.

## [2.0.0] - 2026-10-19

Complete rewrite in Go. ppm now ships as a single binary and no longer needs
`pip install ppm3`.

### Added
- Templates for `ppm init`: `app`, `cli`, `fastapi`, `flask` (`-t`).
- `ppm init` imports an existing `requirements.txt`.
- `ppm add` / `ppm add -D` record direct and dev dependencies in `ppm.yml`.
- `ppm install --prod`, `--clean`, `-r requirements.txt`.
- Automatic `uv` support for much faster installs (`installer:` in `ppm.yml`).
- `ppm exec`, `ppm shell`, `ppm script`, `ppm env`, `ppm use`, `ppm clean`,
  `ppm doctor`, `ppm self install|update|uninstall`.
- Releases are built and published automatically on every push to `main`.
- `pre<name>` / `post<name>` script hooks; extra arguments are passed to scripts.
- Commands work from any subdirectory; global `-C <dir>` flag.
- Shell completion, man pages and `.deb`/`.rpm`/`.apk` packages.

### Changed
- `ppm.yml` uses a flat format. Old ppm3 files are converted automatically.
- The default script is `start` (ppm3's `run` script is migrated).
- `ppm list` shows project dependencies; `ppm list --all` shows everything.

### Fixed
- Virtual environment creation crashed (`_create_virtual_environment` was
  called with an unexpected argument).
- Scripts containing quotes failed because they were wrapped in `'...'`.
- `ppm run` buffered output, lost the exit code, ignored extra arguments and
  broke interactive programs.
- `.env` was never loaded when running scripts.
- `ppm.yml` stored every installed package instead of direct dependencies.
- `git init` committed `.venv` because no `.gitignore` was created.
- Default license typo `ICS` and new projects defaulting to version `1.3.1`.

## [1.3.1] - 2025-04-16

Last release of the Python implementation (`ppm3` on PyPI).

[Unreleased]: https://github.com/rahulcodepython/PPM/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/rahulcodepython/PPM/releases/tag/v2.0.0
