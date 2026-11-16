## Why

Errors and diagnostics from the CLI are written with scattered `fmt.Fprintf` calls to ad-hoc writers. As backup and restore gain real behavior, we need one logging mechanism with levels (info, error, debug) that all commands share, instead of each command printing by hand.

## What Changes

- Add [zerolog](https://github.com/rs/zerolog) as a dependency.
- Add a shared logger set up with zerolog's console writer, writing to stderr.
- Convert the error and diagnostic `fmt.Fprint*` calls in `internal/cli` (unknown command, missing command, command failure, flag parse errors, missing required arguments) to the logger.
- Keep help and usage text (general help, per-command help, usage printed after an error) as plain `fmt.Fprintf` output, unchanged.
- Keep the existing `error: ...` message wording, without timestamps.
- Give commands access to the logger so skeleton and future commands log through it rather than printing diagnostics.

Out of scope: log files, JSON output, log level flags (`--verbose`/`--quiet`), and unit tests.

## Capabilities

### New Capabilities
- `console-logging`: How the program writes errors and diagnostics through a shared leveled logger on stderr, while help and usage stay plain output.

### Modified Capabilities

None. The help and error output of `cli-command-parsing` keeps the same observable text and exit codes.

## Impact

- Code: `internal/cli/cli.go` (error and diagnostic `fmt.Fprint*` calls only), new logger setup package, command packages receive the logger; `main.go` wiring.
- Dependencies: adds `github.com/rs/zerolog` to `go.mod`.
- Existing spec `cli-command-parsing` is unaffected as long as output text and exit codes stay the same.
