# console-logging Specification

## Purpose
Defines how butcherbackup writes errors and diagnostics through a single leveled logger, so all commands report failures consistently, while help and usage remain plain text.

## Requirements

### Requirement: Errors through a shared logger
Error and diagnostic messages SHALL be written through one shared logger instead of ad-hoc print calls, and MUST go to standard error.

#### Scenario: Error output
- **WHEN** a command fails or is invoked incorrectly
- **THEN** the error is written to standard error and standard output remains empty

### Requirement: Help and usage are plain output
Help and usage text SHALL NOT be written through the logger and MUST NOT carry timestamps or level prefixes.

#### Scenario: Help output
- **WHEN** the program is run with `help`
- **THEN** the usage and command list are written to standard output as plain text

#### Scenario: Usage after an error
- **WHEN** a command is invoked with a missing or unknown argument
- **THEN** the error line is logged to standard error, followed by the command's plain usage text on standard error

### Requirement: Error message format
Error messages SHALL keep the form `error: <message>` so that scripts and users see the same text as before logging was introduced.

#### Scenario: Unknown command
- **WHEN** the program is run as `butcherbackup frobnicate`
- **THEN** standard error contains `error: unknown command "frobnicate"` followed by the usage, and the exit status is non-zero

#### Scenario: Not implemented command
- **WHEN** a skeleton command is run with valid arguments
- **THEN** standard error contains `error: <command> is not implemented yet` and the exit status is 1

### Requirement: Exit codes unchanged
Introducing the logger SHALL NOT change the program's exit codes: 0 for success and help, 1 for command failure and 2 for usage errors.

#### Scenario: Usage error
- **WHEN** a required argument is missing
- **THEN** the exit status is 2

### Requirement: Logger available to commands
Commands SHALL write their own diagnostics and failures through the same logger and MUST NOT use direct print calls for them.

#### Scenario: Command message
- **WHEN** a command reports a failure
- **THEN** the message is written to standard error through the logger
