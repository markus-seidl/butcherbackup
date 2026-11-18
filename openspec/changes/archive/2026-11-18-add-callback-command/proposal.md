## Why

When dar writes or reads slices for HDDs or tapes, the backup app has to do something between slices: copy a finished slice to its medium, or ask for a different disk or tape. dar can run a command between slices (`-E`) and waits for it, but the app is a separate long-running process, not that command. We need a small command that turns "dar called a hook" into a file the app can see, and keeps dar waiting until the app says it is done.

## What Changes

- Add a `callback` command to the butcherbackup CLI: `butcherbackup callback [options] <notification-file>`.
- The command creates the notification file with a JSON description of the call, then waits until the file no longer exists, then exits with status 0.
- The JSON contains what dar supplies to a hook, except the slice directory: archive base name, slice number, padded slice number, extension and context (`init`, `operation` or `last_slice`). The command receives these as options, filled in by dar's `%b`, `%n`, `%N`, `%e` and `%c` macros. The slice directory (`%p`) is deliberately left out: the app already knows where it writes the archive, and a path is the one value that can contain spaces or quotes and break the hook command line.
- The app watches for the file, processes the slice, and deletes the file to let dar continue.
- Optional timeout and poll interval, safe handling of an already existing notification file, and signal handling.
- Tests, including an end-to-end test with the real dar, and a README section with an example `-E` command.

Out of scope: the app side that watches and processes the file, a builder helper that assembles the `-E` string, other file-watching mechanisms than polling, and slice progress tracking.

## Capabilities

### New Capabilities
- `callback-command`: A CLI command that publishes a JSON notification file and blocks until the file is removed, for use as a dar slice hook.

### Modified Capabilities

None. The command is registered through the existing dispatcher, which already supports adding commands without changing its requirements.

## Impact

- Code: new package `internal/commands/callback`; `main.go` registers the command. No other code changes.
- Docs: `README.md` gains a usage section.
- Dependencies: standard library only (`encoding/json`, `os/signal`, `time`).
- Runtime: dar treats a non-zero exit of the hook as fatal, so this command's failure modes (existing file, timeout, interrupt) abort dar on purpose rather than let it continue unprocessed.
