## Why

butcherbackup currently has only a placeholder `main.go`. The tool is operated through several distinct actions (backup, restore, listing backups, callbacks), so it needs a command line entry point that dispatches to the right action and lets each action define its own arguments.

## What Changes

- Add a command line parser that takes a command name as the first argument and dispatches to a registered command handler.
- Let each command declare and parse its own additional arguments (flags and positional values) independently of other commands.
- Provide a registry so new commands such as `list-backups` and `callback` can be added later without changing the parser.
- Add skeleton `backup` and `restore` commands that are registered and parse their arguments, but do not yet perform real work.
- Replace the placeholder `main.go` with an entry point that calls the parser.
- Print usage and a non-zero exit code for a missing or unknown command, and per-command help for `-h`/`--help`.

Out of scope: real backup/restore logic, the `list-backups` and `callback` commands, and unit tests (not requested for this change).

## Capabilities

### New Capabilities
- `cli-command-parsing`: Dispatching a command line to a named command, with per-command argument parsing, help output and error handling, plus the skeleton `backup` and `restore` commands.

### Modified Capabilities

None.

## Impact

- Code: `main.go` is rewritten; new Go package(s) for the parser and commands under the `butcherbackup` module.
- Dependencies: none, using the Go standard library (`flag`).
- No existing specs or behavior are affected.
