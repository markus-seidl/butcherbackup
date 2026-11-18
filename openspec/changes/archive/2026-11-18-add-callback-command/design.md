# Design

## Context

The CLI dispatches commands through `internal/cli` (`Parser`, `Command` interface, `NewFlagSet`, `Parse`, `RequireFlags`, `UsageError`, exit codes 0/1/2) and logs errors through zerolog on stderr. Commands live in `internal/commands/<name>` and are registered in `main.go`. The dar wrapper (`internal/tools/dar`) can pass an `-E` command string.

Observed in dar 2.8.5:
- The hook is run through a shell once per slice, with the macros `%p` (slice directory, not used here), `%b` (archive base name), `%n` and `%N` (slice number), `%e` (extension) and `%c` (context: `init` when a slice is asked before the catalogue is read, `operation` for normal slice handling, `last_slice` after the last slice has been written when creating), and `%%` for a literal percent. These are the only values dar supplies: no environment variables with dar information are set.
- dar waits for the hook to finish. A non-zero exit aborts dar (exit 6, "Fatal error on user command line"). When creating, the hook runs after each finished slice with context `operation`, and after the last one with `last_slice`. When reading, the first call has context `init` (slice asked before the catalogue is read), then `operation`.
- The hook has no terminal on standard input.
- When a hook fails, dar aborts but still closes the archive by writing a tiny terminator slice (about 100 bytes, the next slice number), so "dar stopped" means the next slice holds no data, not that its file is absent.
- `%N` equalled `%n` in all runs here; dar documents it as the zero-padded form, so both are passed through as given.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- A blocking handshake between dar and the app that needs only a file system: the command creates a file, the app deletes it.
- Every failure of the handshake stops dar.

**Non-Goals:**
- The app side (watching, copying, mounting media).
- Building the `-E` string in the dar builder.
- Other signalling mechanisms (sockets, FIFOs, inotify).

## Decisions

- **Package `internal/commands/callback`** with the `Command` type following the existing command pattern (`New(log)`, `Name`, `Description`, `Run(args)`), registered in `main.go`. The notification and wait logic sit in unexported functions taking a `context.Context`, so tests can drive them without signals or real time.
- **Information arrives as options, not by parsing dar's text.** dar substitutes macros into the command string, so the hook passes them as `--basename %b --number %n --number-padded %N --extension %e --context %c`. The slice directory macro `%p` is intentionally not used: it is the only value that can contain spaces or quotes, which dar expands textually into a shell command, and the app knows where it writes the archive anyway. Without it the hook line needs no quoting. The notification file is the last argument because the standard `flag` package stops parsing options at the first positional argument; any further argument is a usage error. Alternative: reading values from environment variables, rejected because dar sets none.
- **JSON object** with the keys `basename`, `number` (integer), `number_padded` (string, to keep leading zeros), `extension`, `context`. Keys for options that were not given are omitted, using `omitempty` and a pointer for the number. Marshalled with `encoding/json`, indented, newline-terminated, so values with spaces, quotes or non-ASCII text are always valid JSON. The context value is not validated against `init`/`operation`/`last_slice`, so future dar contexts keep working.
- **Atomic creation:** check that the path does not exist (`os.Lstat`), write the content to a temporary file `.<name>.<pid>.tmp` in the same directory, then `os.Rename` it to the final name. A reader polling the path sees nothing or the full document. Alternatives: `O_EXCL` create and write in place, rejected because a reader can see a partial file; `os.Link` for an exclusive atomic create, rejected because hard links are not supported on FAT/exFAT, which are common on external HDDs. The check-then-rename leaves a tiny window in which another creator could be overwritten; dar runs hooks one at a time, so this is accepted.
- **Existing file is an error.** An existing notification file means a previous handshake was not finished, or another callback is running. Overwriting could hide an unprocessed slice, so the command exits 1 and does not touch the file. A stale file left by a crash has to be removed by the app. Alternative: overwrite, rejected for the reason above.
- **The directory must exist.** Creating directories would hide a configuration mistake and could create them with unexpected permissions.
- **Waiting by polling `os.Lstat`** every `--poll-interval` (default 200 ms) until it reports "does not exist". Standard library only and works on any file system, including network and exFAT, where change notifications are unreliable. The first check happens immediately after creation. Any other stat error is returned as a failure. Alternative: inotify/fsnotify, rejected as a dependency and platform-specific for a handshake that tolerates a 200 ms delay.
- **Timeout and signals:** `--timeout` (duration, default none) bounds the wait; `signal.NotifyContext` for SIGINT and SIGTERM stops it. In both cases the command exits 1 and leaves the file in place: it cannot know whether the app is mid-way through processing, and removing the file would tell the app it can proceed. Default is no timeout at all.
- **Exit codes** follow the CLI: invalid arguments are a `UsageError` (2: missing file, extra arguments, invalid number, invalid duration), every runtime failure is a plain error (1), success is 0. The command prints nothing on success, so the hook adds no noise to dar's output.
- **Testing:** unit tests with temporary directories for notification content, atomicity of creation and the existing-file rule; wait tests with short intervals using goroutines that delete the file; command tests that call `Run` with arguments. An end-to-end test builds the binary with `go build`, runs real dar through the dar builder's `ExecuteBetweenSlices` with the callback command, and plays the app: it watches for the file, reads the JSON, checks that dar has not started the next slice yet, and deletes the file.

## Risks / Trade-offs

- [A base name with spaces or shell characters would break the unquoted `%b` in the hook line] → The app chooses the base name, so it is documented that it must consist of plain characters (letters, digits, `-`, `_`, `.`). `%n`, `%N`, `%e` and `%c` are always plain.
- [A stale notification file blocks all following slices and aborts dar] → Intended, and the error message names the file, so the cause is clear.
- [Polling adds up to one interval of latency per slice] → 200 ms against slices of many megabytes; configurable.
- [Waiting forever by default] → A hook is only useful if the app responds; use `--timeout` when the app may be gone.
- [Rename may not be atomic on some network file systems] → The notification path should be on a local file system; documented.
- [dar passes no information beyond the macros] → The JSON carries exactly that; anything else (for example the slice directory or the full slice path) is known to or derived by the app from `basename`, `number` and `extension`.
