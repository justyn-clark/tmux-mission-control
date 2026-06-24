# Distribution

`tmc` is a Go command-line application. The primary distribution unit is a single `tmc` binary plus the documented runtime dependency on `tmux`.

The repository identity is the personal GitHub account:

```text
github.com/justyn-clark/tmux-mission-control
```

The Git remote should point at:

```text
git@github.com:justyn-clark/tmux-mission-control.git
```

## Install Paths

### Go Install

For Go users:

```bash
go install github.com/justyn-clark/tmux-mission-control/cmd/tmc@latest
```

Use a specific version when reproducibility matters:

```bash
go install github.com/justyn-clark/tmux-mission-control/cmd/tmc@v0.1.0
```

### GitHub Releases

Tagged releases build archives for:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

Each archive contains:

- `tmc`
- `README.md`

The release also publishes `SHA256SUMS`.

### Local Release Build

Build release artifacts locally:

```bash
scripts/build-release.sh v0.1.0
```

Artifacts are written to `dist/`.

### Homebrew

Homebrew is a good later distribution path for macOS users:

```bash
brew install justyn-clark/tap/tmc
```

That requires a separate tap formula after the first stable GitHub release exists.

## Release Workflow

The GitHub Actions release workflow runs when a tag matching `v*` is pushed:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow:

1. Installs Go and Node.
2. Runs `make format-check`.
3. Runs `go test -count=1 ./...`.
4. Builds macOS and Linux archives.
5. Publishes or updates a GitHub release with archives and checksums.

## Runtime Requirements

Published binaries still require:

- `tmux` 3.x or newer
- A POSIX shell
- A Unix-like host environment

Windows is not a primary target because `tmux` is not a native Windows runtime dependency.
