# tmux-mission-control

`tmc` is a terminal-first workspace launcher and session manager for projects, services, logs, and agent loops built on `tmux`.

It gives a project one human-readable YAML manifest and turns that manifest into a repeatable tmux session with windows, pane layouts, commands, hooks, status inspection, doctor checks, and dry-run output.

## Why

- One command boots a named workspace instead of rebuilding it from memory.
- Layouts are explicit, reviewable, and reusable across repos.
- Startup and shutdown hooks are on disk, not trapped in shell history.
- Dry-run prints the exact actions before anything mutates tmux state.

## Requirements

- Go 1.25 or newer to build `tmc`
- `tmux` 3.x or newer at runtime
- A POSIX shell such as `zsh`, `bash`, or `sh`
- Node.js and npm when running Biome-based repository checks

## Install

```bash
go build -o ./bin/tmc ./cmd/tmc
```

## Development

Bootstrap the local toolchain once, then run the standard checks:

```bash
make bootstrap
make format
make format-check
go test ./...
```

- Go code uses `gofmt`
- JSON or future JS or TS repo assets use `Biome`
- `make bootstrap` installs npm dependencies from `package-lock.json` and may need registry access on a fresh machine; `make format-check` uses the repo-local Biome binary and does not install tools

## Quick Start

Generate a starter manifest for a repo:

```bash
./bin/tmc init --root /absolute/path/to/repo --output project.yml --layout dev
```

Layout-specific scaffolds are opinionated on purpose:

- `dev` -> editor, shell, tests, logs with Go-friendly defaults
- `backend` -> editor, shell, server, logs
- `frontend` -> editor, shell, dev server, tests, logs with npm defaults
- `ops` -> shell, service, logs, docs
- `agent-lab` -> editor, shell, tests, logs, agent, docs

Preview the exact actions:

```bash
./bin/tmc dry-run --file project.yml
```

Validate dependencies and manifest references:

```bash
./bin/tmc doctor --file project.yml
```

Start the workspace:

```bash
./bin/tmc start --file project.yml
```

Inspect or tear down the session:

```bash
./bin/tmc status --session my-session
./bin/tmc stop --session my-session
```

Machine-readable output is available for automation:

```bash
./bin/tmc dry-run --file project.yml --json
./bin/tmc list --managed --json
./bin/tmc status --session my-session --json
```

## Commands

- `tmc init`
- `tmc start --file project.yml [--detach]`
- `tmc stop --session NAME`
- `tmc list [--managed] [--json]`
- `tmc status --session NAME [--json]`
- `tmc doctor [--file project.yml]`
- `tmc dry-run --file project.yml [--detach] [--json]`
- `tmc completion [bash|zsh|fish]`

## Manifest Example

```yaml
version: 1
name: "orders-api"
root: "/absolute/path/to/orders-api"
session: "orders-api"
shell: "/bin/zsh"
attach: true
env:
  APP_ENV: "development"
commands:
  server:
    run: "go run ./cmd/orders-api"
  tests:
    run: "watchexec -r -- go test ./..."
  logs:
    run: "tail -F ./tmp/orders-api.log"
startup_hooks:
  - name: "sync deps"
    command: "go mod download"
windows:
  - name: "workspace"
    layout: "backend"
    panes:
      - role: "editor"
        command: "${EDITOR:-nvim} ."
      - role: "shell"
      - role: "server"
        command_ref: "server"
      - role: "logs"
        command_ref: "logs"
```

## Layouts

Built-in layout presets:

- `dev`
- `backend`
- `frontend`
- `ops`
- `agent-lab`

See [docs/layouts.md](docs/layouts.md) for pane-role ordering and ASCII diagrams.

## Docs

- [docs/manifest-spec.md](docs/manifest-spec.md)
- [docs/layouts.md](docs/layouts.md)
- [docs/doctor.md](docs/doctor.md)
- [docs/examples.md](docs/examples.md)
- [docs/STATUS.md](docs/STATUS.md)

## Notes

- `tmux` is a hard dependency. There is no GUI path.
- `tmc` is a workspace launcher, not a process supervisor, health monitor, deployment system, remote session manager, or restart daemon.
- Pane exit codes are tracked in the local user cache directory so `tmc status` can report the last known command exit when a pane command returns.
- `tmc stop` loads shutdown hooks from the manifest path recorded on session creation.
