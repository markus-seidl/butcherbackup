## Why

butcherbackup will use `dar` to create, list and restore archives, including slicing and isolated catalogues for HDD and tape media. Building `dar` command lines by hand with string concatenation is error prone: dar's options are order-insensitive but combination-sensitive (for example `-C` needs `-A`, `-T` only applies to listing), and mistakes only show up when dar runs. A builder lets the rest of the app describe what it wants and get a validated, testable command.

## What Changes

- Add a Go package that builds `dar` invocations with a fluent builder and produces an argument list (and an executable command) without going through a shell.
- Support these dar options: operations `-c` (create), `-x` (extract), `-l` (list), `-C` (isolate catalogue); modifiers `-A` (reference archive or catalogue), `-@` (on-the-fly isolated catalogue), `-R` (root directory), `-z` (compression with optional algorithm and level), `-s` (slice size), `-S` (first slice size), `-v` (verbose), `-affs` (read the archive header from the first slice), `-T` (list format: `normal`, `tree`, `xml`, `slicing`) and the file selection filters `-I` (include mask), `-X` (exclude mask), `-g` (include path) and `-P` (exclude path), each repeatable.
- Validate option combinations when the command is built and return clear errors for invalid ones.
- Allow running the built command through an injectable runner, so it can be tested without dar installed.
- Add unit tests for argument generation and validation, and an integration test that runs the real `dar` when it is installed.

Out of scope: wiring the builder into the `backup`/`restore` commands, parsing dar output, other dar options (encryption, regex masks `-ar`, `-t`, `-d`, `-+`, `-E`, `-p`), and dar_manager or dar_split.

## Capabilities

### New Capabilities
- `dar-command-builder`: Building validated `dar` command lines from a typed description of the operation and options.

### Modified Capabilities

None.

## Impact

- Code: new package `internal/tools/dar` (builder, validation, runner) with tests. No change to existing packages.
- Dependencies: none beyond the Go standard library (`testing`, `os/exec`).
- Runtime: the real `dar` binary is only needed for the optional integration test.
