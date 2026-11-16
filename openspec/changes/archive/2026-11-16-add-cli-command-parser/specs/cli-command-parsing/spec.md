# Spec Delta

## Purpose
Defines how the butcherbackup command line selects a command and passes it its own arguments, so that backup, restore and future commands share one consistent entry point.

## ADDED Requirements

### Requirement: Command dispatch
The program SHALL treat the first argument as the command name and run the matching command, passing it all remaining arguments unchanged.

#### Scenario: Known command
- **WHEN** the program is run as `butcherbackup backup --source /data`
- **THEN** the `backup` command runs and receives `--source /data` as its arguments

#### Scenario: Missing command
- **WHEN** the program is run with no arguments
- **THEN** it prints the general usage including the list of available commands to standard error and exits with a non-zero status

#### Scenario: Unknown command
- **WHEN** the program is run as `butcherbackup frobnicate`
- **THEN** it prints an error naming `frobnicate`, the list of available commands, and exits with a non-zero status

### Requirement: Per-command arguments
Each command SHALL define and parse its own arguments, and arguments of one command MUST NOT be accepted or interpreted by another command.

#### Scenario: Valid arguments
- **WHEN** a command is run with arguments it defines
- **THEN** the command receives the parsed values

#### Scenario: Unknown argument
- **WHEN** a command is run with an argument it does not define
- **THEN** the program prints an error and that command's usage, and exits with a non-zero status without running the command

#### Scenario: Missing required argument
- **WHEN** a command is run without an argument it requires
- **THEN** the program prints an error naming the missing argument and that command's usage, and exits with a non-zero status

### Requirement: Help output
The program SHALL provide general help and per-command help.

#### Scenario: General help
- **WHEN** the program is run with `-h`, `--help` or `help`
- **THEN** it prints the usage with the available commands and a one-line description of each, and exits with status zero

#### Scenario: Command help
- **WHEN** a command is run with `-h` or `--help`
- **THEN** it prints that command's description and arguments and exits with status zero

### Requirement: Extensible command set
New commands SHALL be addable by registering them, without changing how existing commands are parsed or dispatched.

#### Scenario: Added command
- **WHEN** a new command is registered
- **THEN** it appears in the general help and can be invoked by name

### Requirement: Skeleton backup command
The `backup` command SHALL be available and accept a source path, a destination path and the arguments needed to select what to back up, but MAY perform no backup yet.

#### Scenario: Backup invoked
- **WHEN** `butcherbackup backup --source /data --destination /mnt/disk1` is run
- **THEN** the arguments are parsed and the command reports that backup is not implemented yet, exiting with a non-zero status

### Requirement: Skeleton restore command
The `restore` command SHALL be available and accept a backup location, a target path and an optional list of files to restore, but MAY perform no restore yet.

#### Scenario: Restore invoked
- **WHEN** `butcherbackup restore --backup /mnt/disk1 --target /restore` is run
- **THEN** the arguments are parsed and the command reports that restore is not implemented yet, exiting with a non-zero status
