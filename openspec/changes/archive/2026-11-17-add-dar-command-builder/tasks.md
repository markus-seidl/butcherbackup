# Tasks

## 1. Core types and argument generation

- [x] 1.1 Create `internal/tools/dar` with the `Operation`, `Compression`, `ListFormat` and `Size` types, `ParseSize`, and the `Builder` with chained setters for operations, `-A`, `-@`, `-R`, `-z`, `-s`, `-S`, `-v`, `-affs` and `-T`. Verify: `go build ./...` succeeds.
- [x] 1.2 Implement `Build()` returning a `*Command` whose `Args()` emits the fixed order (operation, `-A`, `-@`, `-R`, `-z`, `-s`, `-S`, `-affs`, `-v`, `-T`), with `-z` and `-T` attached to their values. Add table-driven tests in `internal/tools/dar/builder_test.go` for each option alone and combined, including paths with spaces and repeated builds. Verify: `go test ./internal/tools/dar/...` passes.

## 2. Validation

- [x] 2.1 Implement `validate()` for operations: none, more than one, and empty archive name, returning errors joined with `errors.Join`. Add tests for each error message. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 2.2 Validate option values and compatibility: compression algorithm, slice size format, `-S` requires `-s`, slicing only for create/isolate, `-T` format values and only for list, `-C` requires `-A`, `-@` only for create, `-R` only for create/extract. Add a test per rule and one test showing several errors reported together. Verify: `go test ./internal/tools/dar/...` passes.

## 3. Running

- [x] 3.1 Add the `Runner` interface, `Result`, the `exec`-based default runner, `Binary(path)` and `Command.Run(ctx, runner)`. Add tests with a fake runner covering the default `dar` program, a custom binary, returned output, and a non-zero exit returning the code and stderr. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 3.2 Add an integration test in `internal/tools/dar/integration_test.go` that skips when `dar` is not on `PATH` and otherwise, in `t.TempDir()`, creates an archive with `-s` and `-z`, lists it with `-Tslicing`, isolates it with `-C -A`, and extracts the archive using `-x -A <catalogue> -affs` (the builder has no `-g` filter, so the whole archive is extracted), checking file contents. Verify: `go test ./...` passes, and the integration test runs rather than skips in this environment (`go test -v -run Integration ./internal/tools/dar/`).
- [x] 3.3 Run the full checks. Verify: `go vet ./...` is clean, `gofmt -l .` prints nothing and `go test ./...` passes.

## 4. File and path filters

- [x] 4.1 Add `IncludeMask`, `ExcludeMask`, `IncludePath` and `ExcludePath` (variadic, order kept) to the builder and emit `-I`, `-X`, `-g`, `-P` after `-T`. Add argument tests for each filter alone, repeated filters keeping order, all four together, and masks and paths with spaces or wildcards staying single arguments. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 4.2 Validate filters: empty values, absolute `-g`/`-P` paths, and any filter on isolate. Add one test per rule and extend the combined-errors test. Verify: `go test ./internal/tools/dar/...` passes.
- [x] 4.3 Extend `internal/tools/dar/integration_test.go`: create an archive with `-I`/`-X` and check the listing, create with `-g`/`-P` and check the listing, and extract a single file with `-g` into a temporary directory checking only that file is restored. Verify: `go test -v -run Integration ./internal/tools/dar/` passes and does not skip.
- [x] 4.4 Run the full checks. Verify: `go vet ./...` is clean, `gofmt -l .` prints nothing and `go test ./...` passes.
