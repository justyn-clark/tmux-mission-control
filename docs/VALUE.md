# Value

`tmc` turns local workspace startup into a repository artifact.

Without `tmc`, a project workspace often lives in memory, shell history, tmux muscle memory, or chat instructions. With `tmc`, the workspace is described in a manifest that can be reviewed, copied, versioned, tested, and handed to another human or agent.

## What It Provides

- A repeatable way to open project panes, services, logs, docs, tests, and agent loops.
- A dry-run mode that shows planned tmux mutations before they happen.
- Doctor checks for local dependencies, paths, command heads, and log references.
- Status output for managed sessions, including JSON for automation.
- Explicit startup and shutdown hooks.
- Preset layouts that keep workspace shapes consistent across repositories.

## Why It Matters

The value is not that `tmc` starts tmux. The value is that it makes the expected local operating context durable.

That matters for:

- Long-lived projects that are resumed after weeks or months.
- Multi-service repos where startup order and pane layout matter.
- Agent workflows that need explicit state instead of chat-only instructions.
- Teams that want a shared local workspace contract.
- Auditable project operations where setup should be visible in source control.

## What It Is Not

`tmc` is not a service supervisor, deployment system, health checker, restart daemon, or remote control plane. It intentionally stays small: local manifests in, tmux workspace out.

That boundary keeps the project easy to reason about and makes it a reliable foundation for future tools.
