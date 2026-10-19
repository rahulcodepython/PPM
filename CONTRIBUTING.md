# Contributing to ppm

Thanks for helping! Bug reports, ideas and pull requests are all welcome.

## Reporting bugs and ideas

Open an issue at https://github.com/rahulcodepython/PPM/issues. For bugs,
include the output of `ppm doctor` and the exact command you ran.

## Development setup

You need Go (see `go.mod` for the version) and Python 3.

```bash
git clone https://github.com/rahulcodepython/PPM.git
cd PPM
go run ./cmd/ppm --help     # run without installing
make test                   # unit tests   (go test ./...)
make lint                   # gofmt + vet
make dev                    # build and install your copy as `ppm`
```

See [Build from source](README.md#build-from-source) for Windows commands.

## Project layout

| Path | What lives there |
| --- | --- |
| `cmd/ppm` | `main` package |
| `internal/cli` | cobra commands, one file per area |
| `internal/project` | `ppm.yml` loading, saving, requirement parsing |
| `internal/python` | finding Python, venv creation, pip/uv calls |
| `internal/scaffold` | `ppm init` templates |
| `internal/dotenv` | `.env` reading and writing |
| `internal/ui` | colours and prompts |
| `tools/gendocs` | generates man pages and shell completions for releases |
| `.github/scripts` | release version calculation |

## Pull requests

1. Fork and create a branch from `main`.
2. Keep changes focused; one feature or fix per PR.
3. Run `gofmt -w .`, `go vet ./...` and `go test ./...`.
4. Add a line under "Unreleased" in `CHANGELOG.md`.

Use [Conventional Commits](https://www.conventionalcommits.org/) messages.
Every merge to `main` is released automatically, and the commit messages
set the version and the release notes:

| Message | Release |
| --- | --- |
| `fix: ...`, `chore: ...`, `refactor: ...` | patch (2.0.0 to 2.0.1) |
| `feat: ...` | minor (2.0.1 to 2.1.0) |
| `feat!: ...` or a `BREAKING CHANGE:` line | major (2.1.0 to 3.0.0) |
| contains `[skip release]`, or only docs changed | no release |

By contributing you agree that your work is released under the
[MIT License](LICENSE) and that you follow the
[Code of Conduct](CODE_OF_CONDUCT.md).
