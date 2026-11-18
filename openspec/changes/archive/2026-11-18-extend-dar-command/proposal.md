## Why

The dar wrapper can build commands, but callers still cannot follow a running dar or decide which media they need. For HDD and tape backups we need to show which file is being processed, surface dar's errors, request only the slices that hold the files to restore, and let dar call a hook when it needs a new slice. The capability name `dar-command-builder` also no longer fits, since the wrapper now covers more than building.

## What Changes

- **Rename** the capability `dar-command-builder` to `dar-command`. All existing requirements move over unchanged except where extended below. The Go package stays `internal/tools/dar`.
- **Builder:** add `-E <command>` (run between slices) and `-F <command>` (same, for the reference archive). `-F` is only accepted together with `-A`.
- **Runner:** `ExecRunner` can optionally forward stdout and stderr to writers while the command runs, in addition to capturing them, so output can be parsed live.
- **Verbose output parser** for `-v` output: keeps only the latest file being worked on (create and extract) and collects errors reported by dar. It accepts lines from stdout and stderr.
- **Slicing listing parser** for `-l -Tslicing` output: parses entries with their slice ranges, and given a regular expression matched against the file name, returns the slices that must be provided.
- Tests for all of the above, including integration tests against the real dar, and fixtures captured from dar 2.8.5.

**Finding, not implemented:** `-v` does not report which slice is being written. I tried `-v`, `-vs`, `-vt`, `-vd`, `-vm`, `-vf` and `-va` for create and extract with sliced archives, and no line names a slice. As requested, slice progress is therefore not part of the verbose parser. dar does expose the slice number through the `-E` hook (`%n`, plus `%c` for the context), which this change adds to the builder; a later change can use it for slice progress.

Out of scope: slice progress tracking, parsing the final statistics, `-t`/`-d` output, XML listings, and wiring any of this into the `backup`/`restore` commands.

## Capabilities

### New Capabilities
- `dar-command`: Building validated dar command lines, running them, parsing verbose progress and errors, and finding the slices needed for files from a slicing listing. Replaces `dar-command-builder`.

### Modified Capabilities

None. `dar-command-builder` is retired by the rename: its delta removes all requirements, and `dar-command` carries them forward.

## Impact

- Code: `internal/tools/dar` gains hook options, runner stream forwarding, a verbose parser and a slicing parser, with tests and `testdata` fixtures. No change to other packages.
- Specs: `openspec/specs/dar-command-builder` is removed and `../../../specs/dar-tool` is created at archive time.
- Dependencies: standard library only (`regexp`, `bufio`, `sync`).
