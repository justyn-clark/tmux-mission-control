# Manifest Spec

`tmc` reads a single YAML document with version `1`.

## Top-Level Fields

| Field | Type | Required | Notes |
|------|------|----------|------|
| `version` | integer | no | Defaults to `1` |
| `name` | string | yes | Project name |
| `root` | string | yes | Project root. Relative paths resolve from the manifest file directory |
| `session` | string | no | Defaults to a sanitized form of `name` |
| `shell` | string | no | Defaults to `$SHELL`, then `/bin/sh` |
| `attach` | boolean | no | Defaults to `true` |
| `env` | map[string]string | no | Applied to hooks and pane commands |
| `commands` | map[string]command | no | Reusable named commands |
| `startup_hooks` | list[hook] | no | Run before tmux session creation |
| `shutdown_hooks` | list[hook] | no | Run during `tmc stop` before `kill-session` |
| `windows` | list[window] | yes | At least one window is required |

## Command Definition

Command definitions can be a string or a map.

String form:

```yaml
commands:
  tests: "go test ./..."
```

Map form:

```yaml
commands:
  server:
    run: "go run ./cmd/api"
    cwd: "./services/api"
    env:
      PORT: "8080"
    description: "Run the local API server"
```

Fields:

| Field | Type | Required | Notes |
|------|------|----------|------|
| `run` | string | yes | Shell command to execute |
| `cwd` | string | no | Relative to project root |
| `env` | map[string]string | no | Merged over top-level and window env |
| `description` | string | no | Documentation only |

## Hook Definition

Hooks can be a string or a map.

String form:

```yaml
startup_hooks:
  - "go mod download"
```

Map form:

```yaml
shutdown_hooks:
  - name: "stop preview tunnel"
    command: "pkill -f cloudflared"
    cwd: "."
    env:
      APP_ENV: "development"
```

Fields:

| Field | Type | Required | Notes |
|------|------|----------|------|
| `name` | string | no | Used for status text |
| `command` | string | yes | Shell command |
| `cwd` | string | no | Relative to project root or window root |
| `env` | map[string]string | no | Merged into hook execution env |

## Window Definition

```yaml
windows:
  - name: "workspace"
    layout: "backend"
    root: "."
    env:
      API_ENV: "dev"
    startup_hooks:
      - "make prepare"
    shutdown_hooks:
      - "make clean-runtime"
    panes:
      - role: "editor"
      - role: "shell"
      - role: "server"
        command_ref: "server"
      - role: "logs"
        log_files:
          - "./tmp/api.log"
```

Fields:

| Field | Type | Required | Notes |
|------|------|----------|------|
| `name` | string | yes | Must be unique in the manifest |
| `layout` | string | no | Defaults to `dev` |
| `root` | string | no | Relative to project root |
| `env` | map[string]string | no | Merged into pane and hook env |
| `startup_hooks` | list[hook] | no | Run before panes are created for that window |
| `shutdown_hooks` | list[hook] | no | Run during `tmc stop` in reverse manifest order |
| `panes` | list[pane] | yes | Order maps onto the selected layout template |

## Pane Definition

| Field | Type | Required | Notes |
|------|------|----------|------|
| `role` | string | no | Defaults from the layout slot if omitted |
| `title` | string | no | Pane title shown inside tmux |
| `cwd` | string | no | Relative to window root |
| `command` | string | no | Inline command |
| `command_ref` | string | no | Reference into top-level `commands` |
| `env` | map[string]string | no | Merged over manifest, window, and command env |
| `log_files` | list[string] | no | If `role: logs` and `command` is empty, `tmc` builds `tail -F ...` automatically |

`command` and `command_ref` are mutually exclusive.

## Supported Pane Roles

- `editor`
- `shell`
- `tests`
- `logs`
- `server`
- `agent`
- `docs`

## Path Resolution Rules

- `root` resolves relative to the manifest file location.
- `window.root`, `command.cwd`, `hook.cwd`, and `pane.cwd` resolve relative to their parent root.
- `pane.log_files` resolve relative to the window root.

## Exit Tracking

When `tmc` dispatches a pane command, it writes the last observed exit code for that pane into the user cache directory. `tmc status` reads that file back to show the last known result.
