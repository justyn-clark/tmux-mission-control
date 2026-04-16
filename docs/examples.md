# Examples

The repository ships five example manifests in [examples](/Users/justin/Documents/Justyn Clark Network/REPOS/tmux-mission-control/examples).

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

## Example Patterns

- Go service: editor, shell, server, logs
- React app: editor, shell, dev server, tests, logs
- Python CLI: editor, shell, tests, docs
- Dual app: frontend and backend windows in one session
- Agent lab: editor, shell, test loop, logs, agent, docs
