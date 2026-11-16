# Design

## Context

Output currently uses `fmt.Fprintf` in `internal/cli/cli.go` (usage, help, errors, missing-argument messages) and `flag.FlagSet` writes its own errors and `PrintDefaults` to `fs.Output()`. The existing spec `cli-command-parsing` requires help on stdout and errors on stderr with specific exit codes. Help and usage text is deliberately kept as plain `fmt.Fprintf`. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Errors and diagnostics go through zerolog.
- Output text stays visually identical to today: `error: ...` for errors, unchanged help and usage.

**Non-Goals:**
- Converting help and usage text to the logger.
- Log levels selectable by flags, file logging, JSON output.
- Changing exit codes or message wording.

## Decisions

- **One stderr logger built on `zerolog.ConsoleWriter`.** Errors only, so a single logger on stderr suffices and the stdout/stderr split of the spec is kept (help stays on stdout via `fmt.Fprintf`). Alternative: separate stdout and stderr loggers, rejected because nothing but help would use stdout.
- **Message-only console format via `ConsoleWriter.PartsOrder` containing only the message field**, which removes timestamp and level so errors still read `error: <message>`. The `error: ` prefix is part of the logged message at the call site. Alternative: the default format, rejected because it prefixes lines with time and level, changing the wording the spec preserves.
- **Level mapping:** failures use `Error()`. Debug and info levels are available for later use but nothing needs them yet.
- **Package `internal/logging`** exposes `New(w io.Writer) zerolog.Logger`. `cli.Parser` receives the logger in `NewParser`, keeping the writer injectable.
- **Flag output:** each `flag.FlagSet` is set to `io.Discard` while parsing so `flag` does not print its own error. The parse error is then logged as `error: <message>` through the logger and the usage text is printed with `fmt.Fprintf` to stderr. `-h` still prints usage (plain) and exits 0.
- **Commands** receive the logger via their constructors (`backup.New(log)`), replacing direct prints for diagnostics.

## Risks / Trade-offs

- [`ConsoleWriter` may reformat error messages] → Compare error output against the current text during verification.
- [Hiding timestamp and level loses information for diagnostics] → Acceptable now because this is an interactive CLI; revisit when `--verbose` or file logging is added.
- [Error line and plain usage text come from two mechanisms and could interleave] → Both write to stderr from one goroutine in order; verified in task 2.2.
- [Adds a third-party dependency] → zerolog is small and widely used; the change is isolated to `internal/logging`.
