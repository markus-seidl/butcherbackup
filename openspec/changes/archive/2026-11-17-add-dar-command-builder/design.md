# Design

## Context

The module `butcherbackup` (Go 1.27) has `internal/cli`, `internal/commands/*` and `internal/logging`; nothing calls `dar` yet. The installed dar is 2.8.5. Findings from earlier experiments with it that shape the design:
- `-c`, `-x`, `-l`, `-C` and `-@` each take an archive base name as their argument; `-A` takes the reference archive or catalogue; `-R` and `-s`/`-S` take a value.
- `-z[algorithm[:level]]` and `-T<format>` are written as one argument (`-zzstd:3`, `-Tslicing`).
- `-affs` makes dar read the archive header from the first slice, so an isolated catalogue given with `-A` avoids requesting the last slice.
- dar prints a "No terminal found" notice and aborts prompts when there is no tty, which is acceptable for a non-interactive wrapper.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- A typed, fluent builder that cannot silently produce invalid combinations.
- Arguments returned as `[]string` for `exec.Command`, never a shell string.
- Fully testable without dar installed.

**Non-Goals:**
- Parsing dar output, progress reporting, interactive prompts.
- Regex masks (`-ar`) and masks over full paths (`-ap`); only the default glob behavior is supported.
- Options beyond the list in the proposal; the builder must make adding them easy but does not include them.
- Integrating with `backup` and `restore` commands.

## Decisions

- **Package `internal/tools/dar`** with a `Builder` and a `Command`. `Builder` collects settings through chained methods and `Build()` validates and returns `(*Command, error)`. Alternative: functional options (`NewCommand(op, opts...)`), rejected because the request asks for a builder and chaining reads well for many optional settings.
- **Operations as an enum** (`Create`, `Extract`, `List`, `Isolate`), set through `Create(archive)`, `Extract(archive)`, `List(archive)` and `Isolate(archive)` methods that record the operation and archive. Calling two of them is detected at `Build()` and reported as an error, rather than the second call silently winning, so the spec's "only one operation" rule holds.
- **Errors are collected, not panicked.** Setters never fail; `Build()` returns all validation problems joined with `errors.Join`. This keeps chains simple and gives the caller a full list of mistakes.
- **Typed values for constrained options:** `Compression{Algorithm, Level}` with algorithm constants, `ListFormat` constants (`FormatNormal`, `FormatTree`, `FormatXML`, `FormatSlicing`), and `Size` as a validated string (`ParseSize`) so units such as `100M` or `10Mi` pass through to dar unchanged. Alternative: convert sizes to bytes, rejected because dar does the unit math itself and `-aSI`/`-abinary` change the meaning of suffixes.
- **Fixed argument order:** operation, `-A`, `-@`, `-R`, `-z`, `-s`, `-S`, `-affs`, `-v`, `-T`, then the filters `-I`, `-X`, `-g`, `-P`. dar does not care about order but tests and logs stay stable. Option and value are emitted as two arguments (`-R`, `/workspace`) except `-z` and `-T`, which dar requires attached.
- **Runner interface:** `Runner.Run(ctx, program string, args []string) (Result, error)` where `Result` has `ExitCode`, `Stdout`, `Stderr`. The default implementation uses `os/exec` with `exec.CommandContext`. `Command.Run(ctx, runner)` calls it with the configured program (default `dar`, overridable via `Binary(path)`). Non-zero exit returns the `Result` together with an error wrapping the exit code.
- **Filters** are repeatable and added with `IncludeMask`, `ExcludeMask`, `IncludePath` and `ExcludePath`, each variadic and appending in call order. They are stored in four slices and emitted grouped by kind (all `-I`, then `-X`, `-g`, `-P`), each in insertion order. dar 2.8.5 treats `-I`/`-X` order as irrelevant (checked: swapping them gives the same archive), so grouping loses nothing and keeps output stable.
- **Filter semantics observed in dar 2.8.5**, which the validation reflects: masks match file names and do not filter directories (`-I '*.txt'` still stores all directories); `-g` and `-P` take paths relative to `-R` and dar fails with exit 1 on an absolute `-g`, so absolute paths are rejected for both; `-g`, `-P`, `-I` and `-X` work for create, list and extract. dar accepts them with `-C` but its documentation says isolation cannot filter, so isolate is rejected.
- **Compatibility rules** live in one `validate()` function with a table of which options apply to which operation, so adding an option means adding one row.
- **Testing:**
  - Table-driven unit tests for argument output and every validation error.
  - A fake `Runner` that records its inputs, for the run tests.
  - An integration test that builds `-c`, `-l -Tslicing`, `-C -A`, `-x -A -affs` and filtered create/extract (`-I`, `-X`, `-g`, `-P`) against temporary files, skipped with `t.Skip` when `dar` is not on `PATH`.

## Risks / Trade-offs

- [Validation rules may drift from what dar accepts] → The integration test exercises the real binary, and rules are limited to ones observed in dar 2.8.5 (`-C` needs `-A`; `-T` only for listing).
- [Mask and path matching is dar's, not the builder's] → The builder only checks values for emptiness and absolute paths; the integration test checks that filters select the expected files.
- [`-T` and `-affs` were observed with `-l`/`-x` only] → Rules are restrictive on purpose; relax them when another operation needs them.
- [Compression algorithm names depend on dar version (zstd needs 2.7+)] → The builder validates names but cannot check the installed dar; dar's own error surfaces through the runner result.
- [dar prompts when it needs input and there is no tty] → The wrapper treats a non-zero exit as failure; interactive use is out of scope.
