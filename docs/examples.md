# Examples

The repository ships five example manifests in [`examples/`](../examples).

## Included Manifests

- `go-service.yml`
- `react-app.yml`
- `python-cli.yml`
- `dual-app.yml`
- `agent-lab.yml`

## How To Use Them

1. Copy the example that matches the workspace shape you want.
2. Replace `root` with the absolute path to your repo.
3. Adjust commands and log paths to match that repo.
4. Run `tmc doctor --file <example>`.
5. Run `tmc dry-run --file <example>`.
6. Run `tmc start --file <example>`.

If you are starting from scratch instead of copying an example, use `tmc init --layout <layout>` first, then edit the generated manifest.

## Example Patterns

- Go service: editor, shell, server, logs
- React app: editor, shell, dev server, tests, logs
- Python CLI: editor, shell, tests, docs
- Dual app: frontend and backend windows in one session
- Agent lab: editor, shell, test loop, logs, agent, docs
