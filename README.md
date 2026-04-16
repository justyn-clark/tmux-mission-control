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

## Install

```bash
go build -o ./bin/tmc ./cmd/tmc
```

## Development

Formatting is explicit and checked before publish:

```bash
npm install
make format
make format-check
go test ./...
```

- Go code uses `gofmt`
- Python utilities use `ruff format`
- JSON or future JS or TS repo assets use `Biome`

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

## Commands

- `tmc init`
- `tmc start --file project.yml [--detach]`
- `tmc stop --session NAME`
- `tmc list`
- `tmc status --session NAME`
- `tmc doctor [--file project.yml]`
- `tmc dry-run --file project.yml [--detach]`
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

## Notes

- `tmux` is a hard dependency. There is no GUI path.
- Pane exit codes are tracked in the local user cache directory so `tmc status` can report the last known command exit when a pane command returns.
- `tmc stop` loads shutdown hooks from the manifest path recorded on session creation.
