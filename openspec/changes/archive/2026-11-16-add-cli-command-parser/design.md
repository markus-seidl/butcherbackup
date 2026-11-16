# Design

## Context

`main.go` is a placeholder that prints a greeting. The module is `butcherbackup` (Go 1.27) with no dependencies. See proposal.md for motivation. The tool is meant to run as a single binary, possibly from scripts and callbacks, so exit codes and stderr/stdout usage matter.

## Goals / Non-Goals

**Goals:**
- One dispatcher that maps a command name to a handler, with each handler owning its argument parsing.
- A small interface so `list-backups` and `callback` can be added by registering one more value.
- Skeleton `backup` and `restore` commands showing the pattern.

**Non-Goals:**
- Global flags shared across commands (can be added later before the command name).
- Shell completion, config files, or environment variable handling.
- Any real backup or restore behavior.

## Decisions

- **Standard library `flag` with one `FlagSet` per command**, instead of cobra or urfave/cli. It avoids a dependency, and per-command `FlagSet` gives isolated arguments and automatic `-h` output. Trade-off: no built-in subcommand tree or completion, which this project does not need yet.
- **`Command` interface** with `Name()`, `Description()` and `Run(args []string) error`. Each command creates its own `FlagSet` inside `Run`, parses `args`, and validates required values. Alternative considered: a struct with a function field, which is lighter but less clear where flags are declared.
- **Registry** in the parser package (`Register(Command)`, ordered list for help output). `main` registers commands explicitly instead of via `init()` side effects, so the command set is visible in one place.
- **Package layout:** `internal/cli` (dispatcher, registry, help, exit codes) and one package per command: `internal/commands/backup` and `internal/commands/restore`, each with its code in `main.go` (declared as `package backup` and `package restore`). Each command package exports a constructor returning its `Command`. The top-level `main.go` imports both packages, registers them, and calls `os.Exit` with the returned code. Alternative considered: a single `internal/commands` package with one file per command, which is simpler but lets commands share unexported helpers by accident; separate packages keep each command's arguments and logic isolated and give `list-backups` and `callback` a clear place to go.
- **Exit codes:** 0 success or help, 2 for usage errors (unknown command or argument, missing required argument), 1 for command failures, including the "not implemented" skeleton result.
- **Help routing:** `-h`, `--help` and `help` as first argument print general help. Command-level `-h` is handled by `flag.ErrHelp` and treated as success.
- **Skeleton arguments:** `backup` takes `--source` (required), `--destination` (required) and repeatable `--exclude`. `restore` takes `--backup` (required), `--target` (required) and positional file names. These are placeholders, expected to change with real logic.

## Risks / Trade-offs

- *Skeleton flags may not fit the real backup design* → They are isolated in each command file and can change without touching the dispatcher.
- *`flag` stops at the first positional argument, so flags after positionals are ignored* → Document in command usage; revisit if it hurts.
- *No tests in this change, per request* → Verify manually with the commands listed in tasks.md.
