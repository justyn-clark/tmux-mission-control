# Layouts

`tmc` ships with fixed layout presets. Pane order in the manifest maps onto slot order in the selected layout.

## `dev`

Roles by default:

1. `editor`
2. `shell`
3. `tests`
4. `logs`

ASCII shape:

```text
+-----------------------+-------------+
| editor                | shell       |
|                       +-------------+
|                       | tests       |
|                       +-------------+
|                       | logs        |
+-----------------------+-------------+
```

## `backend`

Roles by default:

1. `editor`
2. `shell`
3. `server`
4. `logs`

```text
+-----------------------+-------------+
| editor                | shell       |
|                       +-------------+
|                       | server      |
|                       +-------------+
|                       | logs        |
+-----------------------+-------------+
```

## `frontend`

Roles by default:

1. `editor`
2. `shell`
3. `server`
4. `tests`
5. `logs`

```text
+-----------------------+-------------+
| editor                | shell       |
|                       +-------------+
|                       | server      |
|                       +-------------+
|                       | tests       |
|                       +-------------+
|                       | logs        |
+-----------------------+-------------+
```

## `ops`

Roles by default:

1. `shell`
2. `server`
3. `logs`
4. `docs`

```text
+-----------------------+-------------+
| shell                 | server      |
|                       +-------------+
|                       | logs        |
|                       +-------------+
|                       | docs        |
+-----------------------+-------------+
```

## `agent-lab`

Roles by default:

1. `editor`
2. `shell`
3. `tests`
4. `logs`
5. `agent`
6. `docs`

```text
+-----------------------+-------------+
| editor                | shell       |
|                       +-------------+
|                       | tests       |
+------------+----------+-------------+
| agent      | docs     | logs        |
+------------+----------+-------------+
```

## Notes

- Layouts are deterministic. `tmc` does not infer pane positions from role names.
- If you provide fewer panes than the layout supports, only the leading slots are used.
- If you provide more panes than a layout supports, `tmc` fails validation during planning.
