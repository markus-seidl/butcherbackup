# Tasks

## 1. Parser

- [x] 1.1 Create `internal/cli` with the `Command` interface, a registry (`Register`, ordered listing) and a `Run(args []string) int` dispatcher returning exit codes 0/1/2. Verify: `go build ./...` succeeds.
- [x] 1.2 Implement general help (`-h`, `--help`, `help`) and the missing/unknown command errors with a command list on stderr. Verify: `go run . ` exits 2 with usage; `go run . help` exits 0 listing commands; `go run . frobnicate` exits 2 naming the command.
- [x] 1.3 Treat `flag.ErrHelp` from a command as success (exit 0) and other argument errors as usage errors (exit 2). Verify: `go run . backup -h` exits 0 and `go run . backup --bogus` exits 2.

## 2. Skeleton commands

- [x] 2.1 Add `internal/commands/backup/main.go` with `--source`, `--destination` (required) and repeatable `--exclude`; after parsing, return a "not implemented" error. Verify: `go run . backup --source /a --destination /b` prints "not implemented" and exits 1; omitting `--source` exits 2 naming it.
- [x] 2.2 Add `internal/commands/restore/main.go` with `--backup`, `--target` (required) and positional file names; after parsing, return a "not implemented" error. Verify: `go run . restore --backup /a --target /b file1` prints "not implemented" and exits 1.

## 3. Entry point

- [x] 3.1 Replace the placeholder `main.go` to register `backup` and `restore`, call the dispatcher with `os.Args[1:]` and `os.Exit` with its result. Verify: `go vet ./...` is clean and `go run . help` lists `backup` and `restore` with descriptions.
- [x] 3.2 Add a short usage section to `README.md` showing the command form and the two skeleton commands. Verify: the README examples run as documented.
