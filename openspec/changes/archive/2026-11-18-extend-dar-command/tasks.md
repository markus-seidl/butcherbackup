# Tasks

## 1. Slice hooks (-E, -F)

- [x] 1.1 Add `ExecuteBetweenSlices` (`-E`) and `ExecuteBetweenReferenceSlices` (`-F`) to the builder, emitted last with `-E` before `-F`, value passed as one unmodified argument. Add argument tests, including macros such as `%n` and values with spaces. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 1.2 Validate hooks: empty commands rejected, `-F` requires `-A`. Add one test per rule and extend the combined-errors test. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 1.3 Extend the integration test: run a sliced create with an `-E` hook that appends `%n` and `%c` to a file in `t.TempDir()`, and an extract with `-A <catalogue>` and an `-F` hook, checking the hook files list the slices and contexts. Verify: `go test -v -run Integration ./internal/tools/dar/` passes and does not skip.

## 2. Live output forwarding

- [x] 2.1 Add optional `Stdout` and `Stderr` writers to `ExecRunner`, forwarded while running and still captured in `Result`. Add tests with `sh -c` commands writing to both streams, checking the writers and the result both have the output. Verify: `go test ./internal/tools/dar/...` passes.

## 3. Verbose output parser

- [x] 3.1 Capture real dar 2.8.5 `-v` output into `internal/tools/dar/testdata/` for: create (files, folder, symlink, pipe, hard link), differential create, extract, a restore with corrupted data, a permission-denied save (run as `nobody` through `runuser`, then store the output), a missing archive, and a failing `-E` hook. Verify: each fixture file exists and is non-empty (`ls -l internal/tools/dar/testdata`).
- [x] 3.2 Implement `VerboseParser` current-file tracking (`ParseLine`, `CurrentFile`) and add tests over the fixtures and the spec scenarios for file lines, unrelated lines and paths with spaces. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 3.3 Implement error collection (`Errors`, `DarError`) with the file, restore, not-restored and fatal shapes and fatal deduplication, with tests over the fixtures. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 3.4 Implement `NewWriter` with per-stream partial-line buffering and `Close`, with tests feeding fixtures in odd chunk sizes and from two goroutines. Verify: `go test -race ./internal/tools/dar/...` passes.
- [x] 3.5 Add an integration test running real dar `-v` through `ExecRunner` with the parser's writers on stdout and stderr: after a create the current file is the last `Adding` path; after extracting a corrupted slice an error with the file path is reported; a nonexistent `-R` yields one fatal error. Verify: `go test -v -race -run Integration ./internal/tools/dar/` passes and does not skip.

## 4. Slicing listing parser

- [x] 4.1 Capture a real `-l -Tslicing` listing into `testdata/` from an archive with several slices and entries named with spaces and `: ` (directory, regular files, symlink, named pipe), plus the listing of a differential archive. Implement `ParseSlicing` with `SliceEntry`, `SliceRange` and entry types, with tests over the fixtures and the spec scenarios for single slice, range, no slice, types and non-entry lines. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 4.2 Implement `NeededSlices` and `FindSlices`: full-name regex matching, sorted unique slices, matched entries returned, invalid expressions rejected. Add tests for single and multiple matches, anchored and unanchored patterns, no match, a match without slices and an invalid expression. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 4.3 Add an integration test: create a sliced archive with the builder, list it with `-Tslicing` through the runner, and check `FindSlices` for a large file returns its full slice range while a small file returns only its slice; check against the slice files that exist. Verify: `go test -v -run Integration ./internal/tools/dar/` passes and does not skip.

## 5. Final checks

- [x] 5.1 Run the full checks. Verify: `go vet ./...` is clean, `gofmt -l .` prints nothing and `go test -race ./...` passes.

## Workflow follow-up

- On archive, sync creates `openspec/specs/dar-command` and retires `openspec/specs/dar-command-builder`; confirm `openspec validate --specs` passes and only `dar-command` remains.
