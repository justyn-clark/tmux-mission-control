# Doctor

`tmc doctor` validates the runtime environment and, when a manifest is supplied, the manifest itself.

## Usage

```bash
tmc doctor
tmc doctor --file project.yml
```

## Checks

Without a manifest:

- `tmux` is installed
- the configured shell is available
- an editor executable exists

With a manifest:

- project root exists
- window roots exist
- pane working directories exist
- named commands resolve to an executable head command
- startup hooks and shutdown hooks resolve to an executable head command
- log files exist when `log_files` are configured

## Output

Each line is emitted as:

```text
[PASS] check name: detail
[FAIL] check name: detail
```

`tmc doctor` exits non-zero if any check fails.

## Scope Limits

- Command validation inspects the executable head command, not the full shell graph.
- Hook and pane commands that rely on shell functions or aliases outside the selected shell environment may still fail at runtime.
- `log_files` are only checked when they are explicitly configured.
- Passing doctor means the manifest is plausible for this host. It is not a process health check and does not prove long-running commands will stay alive.
