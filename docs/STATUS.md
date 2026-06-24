# Status

`tmux-mission-control` is a local, terminal-first workspace launcher. It uses a YAML manifest to create repeatable tmux sessions for project workspaces, services, logs, docs, and agent loops.

## Current Shape

- `tmc init` generates preset-based starter manifests.
- `tmc dry-run` prints the planned tmux actions before mutation.
- `tmc doctor` checks local dependencies, paths, and command heads.
- `tmc start` creates managed tmux sessions and records session/pane metadata.
- `tmc list` and `tmc status` inspect tmux state.
- `tmc stop` tears down a session and runs manifest-backed shutdown hooks.
- `tmc version` reports release build metadata.
- Release plumbing builds macOS and Linux archives from tags.

## Product Boundaries

`tmc` is not a process supervisor. It does not restart failed commands, monitor health, deploy services, manage remote machines, or replace tmux itself. It is deliberately CLI-first and local-only.

Preset layouts remain the supported model. Custom layout engines may become valuable later, but the current priority is a small set of predictable layouts that can be reused across repositories.

## Current Limits

- Doctor checks executable heads, not full shell command graphs.
- Runtime status depends on tmux metadata and local cache files.
- A missing recorded manifest during `tmc stop` prevents reliable shutdown-hook execution.
- The CLI has no daemon state; recovery is based on tmux metadata and explicit manifests.
- Integration coverage uses isolated tmux sockets; it does not exercise every possible host tmux configuration.
- Distribution assumes users already have `tmux` installed.

## Roadmap

1. Keep verification deterministic: Go tests, Go formatting, Biome checks, and SMALL strict checks must run without network access.
2. Continue hardening lifecycle execution around partial failures, host tmux differences, and command-level failure reporting.
3. Improve operator output with additional filters and stable JSON fields as automation use cases emerge.
4. Expand isolated tmux integration tests around multi-window sessions, hook failures, and partial cleanup.
5. Add Homebrew tap packaging after the first stable GitHub release.
6. Keep README and manifest docs aligned with actual CLI behavior before adding new features.
