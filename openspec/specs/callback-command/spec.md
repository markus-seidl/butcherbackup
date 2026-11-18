# callback-command Specification

## Purpose
Defines the `callback` command, which publishes a JSON notification file describing a dar slice event and blocks until the file is removed, so dar waits while the backup app handles a slice.

## Requirements

### Requirement: Command and arguments
The program SHALL provide a `callback` command that takes exactly one positional argument, the notification file path, after its options. It SHALL accept the options `--basename`, `--number`, `--number-padded`, `--extension` and `--context`, which carry the values of dar's `%b`, `%n`, `%N`, `%e` and `%c` macros, and the optional `--timeout` and `--poll-interval`.

#### Scenario: Invocation from dar
- **WHEN** dar runs the hook `butcherbackup callback --basename %b --number %n --number-padded %N --extension %e --context %c /run/notify.json` for slice 3 of archive `backup`
- **THEN** the command is started with those values and `/run/notify.json` as the notification file

#### Scenario: Missing notification file
- **WHEN** the command is run without a notification file
- **THEN** it prints an error and its usage and exits with status 2

#### Scenario: Extra arguments
- **WHEN** the command is run with more than one positional argument, such as an option placed after the file
- **THEN** it prints an error and its usage and exits with status 2

#### Scenario: Invalid number
- **WHEN** `--number` is not an integer
- **THEN** it prints an error naming the option and exits with status 2

#### Scenario: Listed in help
- **WHEN** the program is run with `help`
- **THEN** `callback` appears in the command list with a description

### Requirement: Notification file
The command SHALL create the notification file containing a single JSON object with the values it was given: `basename`, `number` (as an integer), `number_padded`, `extension` and `context`. The context SHALL be passed through unchanged, so it carries dar's value (`init`, `operation` or `last_slice`). Options that were not given MUST be omitted. The complete content SHALL be visible as soon as the file exists, so a reader never sees a partial file.

#### Scenario: All values given
- **WHEN** the command is run with basename `backup`, number `3`, padded number `0003`, extension `dar` and context `operation`
- **THEN** the file contains a JSON object with `basename` `backup`, `number` 3, `number_padded` `0003`, `extension` `dar` and `context` `operation`

#### Scenario: Some values omitted
- **WHEN** only `--number 2` is given
- **THEN** the JSON object contains only `number` with the value 2

#### Scenario: Context values
- **WHEN** the command is run with context `init`, `operation` or `last_slice`
- **THEN** the JSON `context` is that same string

#### Scenario: Special characters
- **WHEN** a value contains spaces, quotes or non-ASCII characters
- **THEN** the file contains valid JSON that decodes to the original value

#### Scenario: Never partial
- **WHEN** another process polls the notification path while the command creates it
- **THEN** it either finds no file or finds the complete JSON document

### Requirement: Existing notification file
The command MUST NOT overwrite or remove an existing file at the notification path. It SHALL fail instead. The directory of the notification file MUST already exist and MUST NOT be created by the command.

#### Scenario: File already exists
- **WHEN** the notification file already exists
- **THEN** the command prints an error naming the file, exits with status 1, and leaves the existing file unchanged

#### Scenario: Missing directory
- **WHEN** the directory of the notification file does not exist
- **THEN** the command prints an error naming the directory and exits with status 1

### Requirement: Waiting for removal
After creating the notification file, the command SHALL wait until the file no longer exists and then exit with status 0 without printing anything. Only the absence of the file counts; changes to its content do not.

#### Scenario: File removed by the app
- **WHEN** the app deletes the notification file after some time
- **THEN** the command exits with status 0 shortly after, within about the poll interval

#### Scenario: File removed immediately
- **WHEN** the file is deleted before the command first checks it
- **THEN** the command exits with status 0

#### Scenario: Content rewritten
- **WHEN** the app rewrites the contents of the notification file but does not delete it
- **THEN** the command keeps waiting

#### Scenario: Blocks dar
- **WHEN** dar runs the command as its `-E` hook while creating a sliced archive
- **THEN** dar does not start the next slice until the notification file has been removed

#### Scenario: Quiet on success
- **WHEN** the command exits with status 0
- **THEN** nothing was written to standard output or standard error

### Requirement: Timeout
The command SHALL accept `--timeout` as a duration. When it is set and the file still exists after that time, the command SHALL exit with status 1 and an error message. Without `--timeout` it waits indefinitely. The notification file SHALL be left in place.

#### Scenario: Timeout reached
- **WHEN** `--timeout 1s` is given and the file is not removed within one second
- **THEN** the command exits with status 1, prints that it timed out waiting for the file, and the file still exists

#### Scenario: Invalid duration
- **WHEN** `--timeout` or `--poll-interval` is not a valid positive duration
- **THEN** the command prints an error naming the option and exits with status 2

### Requirement: Interruption
When the command receives an interrupt or termination signal while waiting, it SHALL stop waiting and exit with status 1, leaving the notification file in place.

#### Scenario: Interrupted
- **WHEN** the command is waiting and receives SIGTERM
- **THEN** it exits with status 1 and the notification file still exists

### Requirement: Exit status and dar
The command SHALL exit with status 0 only when the notification file was created and then removed. Every other outcome SHALL be a non-zero status, so dar, which treats a failing hook as fatal, never continues past a slice that was not handled.

#### Scenario: Failure stops dar
- **WHEN** the notification file already exists and dar runs the command as its `-E` hook
- **THEN** the command exits with status 1 and dar aborts instead of continuing with further data slices
