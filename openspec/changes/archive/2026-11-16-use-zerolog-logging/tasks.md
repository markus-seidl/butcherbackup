# Tasks

## 1. Logging package

- [x] 1.1 Add `github.com/rs/zerolog` with `go get github.com/rs/zerolog` and create `internal/logging` with `New(w io.Writer) zerolog.Logger` built on `ConsoleWriter` showing only the message. Verify: `go build ./...` succeeds and `go mod tidy` leaves `go.mod` unchanged.

## 2. Convert the parser

- [x] 2.1 Change `cli.NewParser` to take the logger and convert the error calls in `Parser` (missing command, unknown command, command failure) to it, keeping the `error: ` wording. Help and usage stay `fmt.Fprintf`. Verify: `go vet ./...` clean; `help` prints to stdout unchanged; `frobnicate` prints `error: unknown command "frobnicate"` then usage to stderr only.
- [x] 2.2 Update `NewFlagSet`, `Parse` and `RequireFlags` so flag parse errors and missing-argument errors are logged through the logger while usage text stays `fmt.Fprintf`. Verify: `backup -h` still prints description, usage and defaults; `backup --bogus` and `backup --destination /b` print `error: ...` followed by the usage on stderr.

## 3. Commands and wiring

- [x] 3.1 Pass the logger to `backup.New` and `restore.New` and make `main.go` build the logger and the parser with it. Verify: the build passes and `grep -rn "fmt.Fprint" --include=*.go .` matches only help and usage code in `internal/cli`.
- [x] 3.2 Re-run the exit code checks against a built binary. Verify: no args, unknown command, `--bogus` and missing `--source` exit 2; `help` and `-h` exit 0; valid `backup` and `restore` exit 1 with `error: ... is not implemented yet`; stdout is empty for every error case.
