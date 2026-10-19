# Publishing ppm

Releases are automatic: **every push to `main` that changes code becomes a
new GitHub Release with versioned binaries.** You never build or upload
anything by hand.

## How it works

```
git push origin main
        │
        ▼
.github/workflows/release.yml
  1. test     go vet, go test, go build on Linux, macOS, Windows
  2. version  .github/scripts/next-version.sh reads the commits since the last tag
  3. tag      creates and pushes e.g. v2.1.0
  4. build    GoReleaser (.goreleaser.yaml) builds every binary and package
        │
        ▼
GitHub Release v2.1.0   (release notes start with install instructions)
  ├── ppm_linux_amd64.tar.gz   ppm_linux_arm64.tar.gz
  ├── ppm_darwin_amd64.tar.gz  ppm_darwin_arm64.tar.gz
  ├── ppm_windows_amd64.zip    ppm_windows_arm64.zip
  ├── ppm_2.1.0_amd64.deb / .rpm / .apk   (and arm64)
  └── checksums.txt
        │
        ▼
install.sh, install.ps1 and `ppm self update` download the latest release
```

### How the version is chosen

The commit messages since the last release decide the bump, following
[Conventional Commits](https://www.conventionalcommits.org/):

| Commits contain | Bump | Example |
| --- | --- | --- |
| only `fix:`, `chore:`, `refactor:`, or any other message | patch | 2.0.0 to 2.0.1 |
| at least one `feat:` | minor | 2.0.1 to 2.1.0 |
| `feat!:` / `fix!:` or a `BREAKING CHANGE:` line | major | 2.1.0 to 3.0.0 |

The first release is **v2.0.0**, because ppm3 on PyPI ended at 1.3.1.

**No release** is made when:
- the push only changes Markdown, `docs/`, `LICENSE` or issue templates,
- the head commit message contains `[skip release]`,
- there are no new commits since the last tag, or
- any test fails.

**Manual release:** go to **Actions → release → Run workflow** and pick
`patch`, `minor` or `major`. Use this to force a bump or re-run a release.

If the build fails after tagging, the workflow deletes the tag again. Fix the
problem and push; the same version is reused.

## One-time setup

1. **Push this branch and merge it into `main`.** The install scripts are
   downloaded from `main`, and the first merge triggers v2.0.0.
2. **Let Actions create releases:** go to **Settings → Actions → General →
   Workflow permissions** and choose **Read and write permissions**, then Save.
3. **Protect `main` (recommended):** go to **Settings → Branches → Add rule**
   for `main` and require the `ci` check. Pull requests are then tested
   before they can release.
4. **Description and topics.** In the repo's **About** gear, or:
   ```bash
   gh repo edit rahulcodepython/PPM \
     --description "Python project manager in a single binary: scaffold projects, manage venvs and dependencies (pip/uv), run scripts." \
     --homepage "https://github.com/rahulcodepython/PPM#readme" \
     --add-topic python,cli,golang,project-manager,virtualenv,pip,uv,dependency-management,scaffolding
   ```
5. **Private vulnerability reporting** (used by SECURITY.md): **Settings →
   Code security → Private vulnerability reporting → Enable**.

## Day-to-day workflow

```bash
git checkout -b my-change
# ...edit, then
make lint test
git commit -m "feat: add ppm foo command"     # the prefix decides the version
git push -u origin my-change                  # open a PR; ci.yml tests it
# merge the PR. release.yml publishes v2.x.0 a few minutes later
```

Watch progress in the repo's **Actions** tab. When the run is green, the
release is live for the install scripts and `ppm self update`.

### Test a release build locally

```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser check                     # validate .goreleaser.yaml
make snapshot                        # full build into ./dist, nothing uploaded
./dist/ppm_linux_amd64_v1/ppm version
sh .github/scripts/next-version.sh   # the version the next push would get
```

### Remove a broken release

Only do this right after publishing. If people may have downloaded it,
push a fix instead (it becomes the next patch).

```bash
gh release delete v2.1.0 --yes
git push --delete origin v2.1.0 && git tag -d v2.1.0
```

## Optional: more install channels

### Homebrew (`brew install rahulcodepython/tap/ppm`)

1. Create a public repo `rahulcodepython/homebrew-tap`. It can stay empty.
2. Create a fine-grained personal access token with **Contents: read and
   write** on that repo only.
3. Save it as an Actions secret named `TAP_GITHUB_TOKEN` (**Settings →
   Secrets and variables → Actions**).
4. Uncomment the `homebrew_casks` block in `.goreleaser.yaml`, and add the
   brew line to the README install section.

### Scoop on Windows

Same steps with a repo named `scoop-bucket`, then uncomment the `scoops`
block. Users run
`scoop bucket add ppm https://github.com/rahulcodepython/scoop-bucket` and then
`scoop install ppm`.

### winget

After a release, submit a manifest to
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) with
`wingetcreate new <zip URL>`. Microsoft reviews each submission.

### Go

Nothing to do. `go install github.com/rahulcodepython/PPM/cmd/ppm@latest`
works once a tag exists.

## Retiring ppm3 on PyPI

The old `ppm3` package keeps working for people who have it. To point them
at the new tool, publish a final `ppm3` release from the last Python commit
(`git checkout e551e83`). Start its README with "ppm is now a standalone
binary: see https://github.com/rahulcodepython/PPM", bump the version and
upload with twine as before.

## Code signing (later)

Unsigned binaries work. On first run, Windows SmartScreen may warn about an
"unknown publisher", and macOS may quarantine browser downloads. The install
scripts are not affected. To remove these warnings, add an Authenticode
certificate (Windows) and an Apple Developer ID with notarization (macOS).
GoReleaser supports both through its `signs` and `notarize` sections.
