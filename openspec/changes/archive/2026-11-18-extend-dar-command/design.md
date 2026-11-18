# Design

## Context

`internal/tools/dar` has a `Builder`, `Command`, `Runner` and `ExecRunner` (see the archived change `add-dar-command-builder`). `ExecRunner` buffers stdout and stderr and returns them after dar exits, so nothing can follow a run while it happens. The parsers below read dar 2.8.5 output; the findings here come from running it, not from documentation.

**Observed `-v` output** (stdout; dar prints a "Arguments read from /etc/darrc" notice on stderr):
- Create: `Adding file to archive: <path>`, `Adding folder to archive: <path>`, `Adding symlink to archive: <path>`, `Adding named pipe to archive: <path>`, `Recording hard link into the archive: <path>`, followed by `Saving Filesystem Specific Attributes for <path>`. A differential archive also prints `Adding reference to files that have been destroyed since reference backup...`, which is not a file.
- Extract: `Restoring file's data: <path>` and `Restoring file's FSA: <path>`.
- `-vs`, `-vt`, `-vd`, `-vf` and `-va` only add extra lines such as `Finished Inspecting directory ...`; the file lines come with plain `-v`.
- **No slice information appears for any of `-v`, `-vs`, `-vt`, `-vd`, `-vm`, `-vf`, `-va`**, on create or extract with sliced archives.
- Errors, by shape: `Error while saving <path>: <reason>` (stdout, exit 5); `Error while restoring <path> : <reason>` (note the space before the colon; stdout, exit 5); `<path> not restored (user choice)` (stdout, when dar needs a slice or answer and there is no terminal; exit 0); `Aborting due to exception: <msg>` (stdout) with `FATAL error, aborting operation: <msg>` (stderr) for hard failures, exit 2; `Aborting program. User refused to continue while asking: <msg>` (stderr, exit 4); `All files asked could not be restored` (exit 5 after corrupted data; stdout as `Aborting due to exception: ...` and stderr plain, so the parser reports it as a fatal error through the exception line); a failing `-E` command gives `Aborting due to exception: Fatal error on user command line: <detail>` on stdout and `Aborting program. An error occurred concerning user command execution: Fatal error on user command line: <detail>` on stderr (exit 6). The `-vs`-style summary at the end of a run contains lines such as ` 0 inode(s) not restored (not saved in archive)`, which have a leading space and must not be read as errors.

**Observed `-Tslicing` output:** a header, then one line per entry: `<slices>\t [Data][D]       [EA..][FSA..][S] <10-char permission> <name>`, with the slice column `4`, `1-4` or empty (symlinks, pipes, entries without data in this archive). Names may contain spaces and `: `. Footer lines (`-----`, `All displayed files have their data in slice range [1-4]`) follow. In listings of differential archives, a deleted entry appears as `\t [--- REMOVED ENTRY ----]<name>`, which has no permission field and is not an entry. For an isolated catalogue the slice column refers to the original archive's slices.

**Observed `-E`/`-F`:** the command runs through the shell for each slice with macros expanded (`%n` slice number, `%c` context `init`, `operation` or `last_slice`, `%b`, `%p`, `%e`). It also runs for `-l` and `-x`. `-F` without `-A` only produces a dar warning and exit 0. A failing hook aborts dar with exit 6.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Parsers that work on a finished output and on live output.
- Behavior pinned by real dar output (fixtures) and verified against the real binary.

**Non-Goals:**
- Slice progress, because `-v` does not provide it (see Findings in the proposal).
- Parsing statistics, `-t`/`-d` output, or XML listings.
- Handling terminal prompts; the runner has no tty and dar aborts.

## Decisions

- **Spec rename:** one new capability `dar-command` holds all requirements (existing ones carried over, extended where needed). The old `dar-command-builder` is retired through a delta that removes all its requirements, with `retire_capabilities: true` in the change metadata. The Go package keeps its name `dar`, because the requested rename concerns the spec only.
- **Hooks:** `ExecuteBetweenSlices(command)` (`-E`) and `ExecuteBetweenReferenceSlices(command)` (`-F`) on the builder, emitted last, `-E` before `-F`. The value is passed unmodified as one argument. `-F` requires `-A`, since dar ignores it otherwise and a silent ignore would hide a bug. Alternative: leaving this to dar's warning, rejected because the warning is easy to miss in a wrapper. The caller is responsible for the command content since dar runs it through a shell; no quoting is added.
- **Live output:** `ExecRunner` gets optional `Stdout` and `Stderr` `io.Writer` fields. When set, output is written to them as it is produced and still captured in `Result`, using `io.MultiWriter`. The `Runner` interface stays unchanged. Alternative: a streaming method on the interface, rejected as a larger change for the same benefit.
- **Verbose parser** `VerboseParser`:
  - State: `CurrentFile() string` (latest file only) and `Errors() []DarError`. `DarError` has `Kind` (`file`, `not-restored`, `fatal`), `Path`, `Message` and the original `Line`.
  - Line recognition is by anchored regular expressions compiled once: `^Adding .+ to archive: (.+)$`, `^Recording hard link into the archive: (.+)$`, `^Restoring file's data: (.+)$`; errors: `^Error while \w+ (.+?) ?: (.+)$`, `^(\S.*?) not restored \((.+)\)$` (the non-space start excludes the indented summary lines), and the fatal prefixes `Aborting due to exception: `, `FATAL error, aborting operation: ` and `Aborting program. ` (from which the known sub-prefixes `User refused to continue while asking: ` and `An error occurred concerning user command execution: ` are removed to get the message). `Adding reference to files ...` does not match the first pattern because it has no `to archive:` suffix.
  - FSA lines are not matched because each is preceded by a line naming the same path.
  - Fatal messages are deduplicated: dar prints the same message up to three times (stdout exception, stderr FATAL, stderr abort; the abort line carries a prefix that is removed first), so a fatal with a message already recorded is skipped.
  - Feeding: `ParseLine(string)` for whole lines, and `NewWriter() io.Writer`, which returns a writer with its own partial-line buffer. Several writers (stdout, stderr) share the state under a mutex, so concurrent writes are safe. A trailing partial line is flushed by `Close()` on the writer.
  - The parser can run on a finished `Result` by feeding its stdout and stderr strings.
- **Slicing parser:**
  - `ParseSlicing(io.Reader) ([]SliceEntry, error)` with `SliceEntry{Name string, Type EntryType, Slices []SliceRange}` and `SliceRange{First, Last int}`. Entry lines are recognized by one regular expression: `^([0-9,-]*)\t\s*(?:\[[^\]]*\]\s*){5}(\S{10}) (.*)$`; the permission's first character gives the type (`-` file, `d` directory, `l` symlink, other types as `other`). Everything else is ignored.
  - The name is the rest of the line after the permission and one space, so spaces and colons survive.
  - The slice column accepts comma-separated numbers and ranges, to be tolerant of other layouts, though only single numbers and `a-b` ranges were observed.
  - `NeededSlices(entries, re) (slices []int, matched []SliceEntry)` and a convenience `FindSlices(r io.Reader, pattern string)` that compiles the pattern, returning an error for an invalid one. Matching uses `Regexp.MatchString` on the full listed name, so unanchored patterns match substrings; callers anchor with `^`/`$` as needed.
  - Matching directories: a directory that matches contributes its own slice, which can be a slice a restore of its files would not need. Callers can avoid this by anchoring or by checking `matched` types. Alternative: skipping directories, rejected because it can silently drop slices that hold directory metadata.
  - The result lists only the slices holding the matched data. Opening an archive also reads the first slice header (with `-affs`) or the last slice (the catalogue); the caller adds those according to how it runs dar.
- **Fixtures:** real output from dar 2.8.5 is stored under `internal/tools/dar/testdata/` (create, extract, errors, slicing listing with odd names), and unit tests parse those files. Integration tests then check the same behavior against the live binary.

## Risks / Trade-offs

- [dar message wording may change between versions] → Patterns are anchored on the stable prefixes and covered by fixtures plus the integration tests, which fail on a new dar if wording changes.
- [Unrecognized error shapes are not reported] → The parser documents which shapes it knows; the exit code from `Result` remains the authoritative failure signal. Callers should check both.
- [Filenames containing newlines break line-based parsing] → Not supported; dar's output is line based.
- [`not restored (user choice)` is reported as an error but dar exits 0] → Kept as an error kind of its own so callers can decide.
- [`-E` and `-F` run through a shell] → The builder cannot make that safe; documented, and values come from the caller.
- [Concurrent stdout and stderr writers] → One mutex around parser state, verified with `go test -race`.

## Open Questions

- Whether a later change should track the current slice with an `-E` hook (`%n`, `%c`), since `-v` cannot provide it. This does not affect the work here.
