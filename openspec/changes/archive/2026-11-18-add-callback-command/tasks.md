# Tasks

## 1. Notification file

- [x] 1.1 Create `internal/commands/callback` with the notification type (JSON keys `basename`, `number`, `number_padded`, `extension`, `context`, omitted when not given) and a function that creates the file atomically via a temporary file in the same directory and `os.Rename`, refusing to overwrite an existing path and a missing directory. Add tests: all values, only some values, special characters and non-ASCII round-trip through `json.Unmarshal`, existing file left untouched with an error naming it, missing directory error, no temporary file left behind, and a polling reader that never sees a partial document. Verify: `go test ./internal/commands/callback/...` passes.

## 2. Waiting

- [x] 2.1 Add a wait function taking a context, poll interval and optional timeout that returns once the path does not exist, a timeout error when the time runs out, the context error on cancellation, and any other stat error. Add tests: removal after a delay returns within a few intervals, immediate removal, content rewritten keeps waiting, timeout leaves the file in place, cancellation leaves the file in place. Verify: `go test -race ./internal/commands/callback/...` passes.

## 3. Command

- [x] 3.1 Implement the `Command` (`Name` `callback`, `Description`, `Run`) with options `--basename`, `--number`, `--number-padded`, `--extension`, `--context`, `--timeout`, `--poll-interval`, exactly one positional notification file, usage errors (exit 2) for missing file, extra arguments, a non-integer number and invalid durations, runtime failures as plain errors (exit 1), SIGINT/SIGTERM handling with `signal.NotifyContext`, and no output on success. Verify: `go build ./...` succeeds.
- [x] 3.2 Register the command in `main.go` and add command tests calling `Run` (success with a goroutine removing the file, each usage error, existing file, timeout). Verify: `go test -race ./...` passes, `go run . help` lists `callback`, and with a built binary the exit codes are 2 for `callback` without a file, 1 for an existing file and 1 for `--timeout 300ms` with nobody removing the file.
- [x] 3.3 Add a test that starts the built binary, sends it SIGTERM while it waits, and checks exit status 1 with the file still present. Verify: `go test -race -run Signal ./internal/commands/callback/` passes.

## 4. End to end with dar

- [x] 4.1 Add an integration test that skips without `dar` on `PATH`, builds the binary with `go build` into `t.TempDir()`, creates a sliced archive through `dar.NewBuilder().ExecuteBetweenSlices` using the callback command, and plays the app in a goroutine: wait for the file, decode the JSON, check that the next slice file does not exist yet, record the notification, delete the file. Check that dar succeeds, that the notifications cover slices 1..N in order with context `operation` and `last_slice` for the last, and that `basename`, `extension` and `number` match the slice files on disk. Add a second run that extracts the archive and checks that the first notification has context `init`. Verify: `go test -v -race -run Callback ./internal/tools/dar/ ./internal/commands/callback/` passes and does not skip.
- [x] 4.2 Add a failure case: create the notification file beforehand, run dar with the hook and check that dar exits non-zero (exit 6), reports a fatal user-command error through the verbose parser, and stops after the first slice (slice 2 is at most a tiny terminator, not a data slice). Verify: the test passes with `go test -race ./...`.

## 5. Documentation and final checks

- [x] 5.1 Add a `callback` section to `README.md` with the example dar command (`-E "butcherbackup callback --basename %b --number %n --number-padded %N --extension %e --context %c /run/butcherbackup/slice.json"`), the JSON example, what the three context values mean, the "delete the file to continue" rule, the rule that the base name must use plain characters (letters, digits, `-`, `_`, `.`) and the exit codes. Verify: the README example run against dar with a small sliced archive and a shell loop that deletes the file completes the backup.
- [x] 5.2 Run the full checks. Verify: `go vet ./...` is clean, `gofmt -l .` prints nothing and `go test -race ./...` passes.
