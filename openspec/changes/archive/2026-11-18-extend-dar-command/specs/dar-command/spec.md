# Spec Delta

## Purpose
Defines how butcherbackup describes a dar operation as a validated command line, runs it, follows its verbose output and reads slicing listings to learn which slices a restore needs.

## ADDED Requirements

### Requirement: Operations
The builder SHALL support exactly one operation per command: create (`-c`), extract (`-x`), list (`-l`) or isolate (`-C`), each taking an archive base name.

#### Scenario: Create
- **WHEN** a create operation for archive `/backup/test` is built
- **THEN** the arguments contain `-c /backup/test`

#### Scenario: No operation
- **WHEN** a command is built without an operation
- **THEN** building fails with an error saying an operation is required

#### Scenario: Two operations
- **WHEN** a command is built with both create and extract
- **THEN** building fails with an error saying only one operation is allowed

#### Scenario: Empty archive name
- **WHEN** an operation is given an empty archive base name
- **THEN** building fails with an error naming the empty archive

### Requirement: Supported options
The builder SHALL support these options and emit the matching dar arguments: root directory `-R <path>`, reference archive `-A <archive>`, on-the-fly isolated catalogue `-@ <archive>`, compression `-z`, slice size `-s <size>`, first slice size `-S <size>`, verbose `-v`, first-slice header `-affs`, list format `-T<format>`, the file filters `-I`, `-X`, `-g` and `-P`, and the slice hooks `-E <command>` and `-F <command>`.

#### Scenario: Several options
- **WHEN** a create command is built with root `/workspace`, slice size `100M`, first slice size `200` and compression enabled
- **THEN** the arguments contain `-R /workspace`, `-s 100M`, `-S 200` and `-z`

#### Scenario: Flags and values stay separate
- **WHEN** an archive name or path contains spaces or shell characters
- **THEN** it appears as a single argument, unmodified, and is never interpreted by a shell

### Requirement: Compression
Compression SHALL be optional and accept an algorithm (`gzip`, `bzip2`, `xz`, `zstd`) and an optional level, emitted in dar's `-z[algorithm[:level]]` form.

#### Scenario: Default compression
- **WHEN** compression is enabled without an algorithm
- **THEN** the argument is `-z`

#### Scenario: Algorithm and level
- **WHEN** compression is enabled with algorithm `zstd` and level `3`
- **THEN** the argument is `-zzstd:3`

#### Scenario: Unknown algorithm
- **WHEN** compression is requested with an algorithm that is not supported
- **THEN** building fails with an error naming the algorithm

### Requirement: Slice sizes
Slice sizes SHALL be given as a number with an optional unit suffix (`k`, `M`, `G`, `T`, `P`, `E`, `Z`, `Y`, or the binary forms `ki`, `Mi`, and so on), and are only valid for create and isolate operations.

#### Scenario: Invalid size
- **WHEN** a slice size such as `10 megabytes` or `-5` is given
- **THEN** building fails with an error naming the invalid size

#### Scenario: Slicing on extract
- **WHEN** a slice size is combined with an extract operation
- **THEN** building fails with an error saying slicing applies only to create and isolate

#### Scenario: First slice size without slice size
- **WHEN** a first slice size is set and no slice size is set
- **THEN** building fails with an error saying a slice size is required with a first slice size

### Requirement: List format
The list format SHALL be one of `normal`, `tree`, `xml` or `slicing`, and is only valid for the list operation.

#### Scenario: Slicing format
- **WHEN** a list command is built with format `slicing`
- **THEN** the arguments contain `-Tslicing`

#### Scenario: Unknown format
- **WHEN** a list format other than the supported ones is given
- **THEN** building fails with an error naming the format

#### Scenario: Format on create
- **WHEN** a list format is combined with a create operation
- **THEN** building fails with an error saying the format applies only to list

### Requirement: Option and operation compatibility
The builder SHALL reject options that do not apply to the chosen operation: isolate requires a reference archive, on-the-fly isolation applies only to create, the root directory applies only to create and extract, and the reference slice hook `-F` requires a reference archive.

#### Scenario: Isolate without reference
- **WHEN** an isolate operation is built without a reference archive
- **THEN** building fails with an error saying isolate requires a reference archive

#### Scenario: On-the-fly isolation on extract
- **WHEN** an on-the-fly isolated catalogue is combined with an extract operation
- **THEN** building fails with an error saying it applies only to create

#### Scenario: Reference hook without reference
- **WHEN** a reference slice hook is combined with no reference archive
- **THEN** building fails with an error saying the reference hook requires a reference archive

#### Scenario: Reference with extract
- **WHEN** a reference archive is combined with extract
- **THEN** the arguments contain `-A <archive>` so an isolated catalogue can be used in place of the archive's own catalogue

### Requirement: File and path filters
The builder SHALL support the include mask `-I <mask>`, exclude mask `-X <mask>`, include path `-g <path>` and exclude path `-P <path>`. Each filter MAY be given several times, and the values SHALL be emitted in the order given. Masks match file names. Paths are relative to the root directory, so absolute paths MUST be rejected. Filters apply to create, extract and list, and MUST be rejected for isolate.

#### Scenario: Include path on extract
- **WHEN** an extract command is built with root `/restore` and include path `tools/dev-vm/dev_vm.smolfile`
- **THEN** the arguments contain `-g tools/dev-vm/dev_vm.smolfile`

#### Scenario: Several filters
- **WHEN** a create command is built with include mask `*.txt`, exclude mask `tmp*`, include path `docs` and exclude path `docs/old`
- **THEN** the arguments contain `-I *.txt`, `-X tmp*`, `-g docs` and `-P docs/old`

#### Scenario: Repeated filter keeps order
- **WHEN** include paths `a` and `b` are added in that order
- **THEN** the arguments contain `-g a` before `-g b`

#### Scenario: Masks stay single arguments
- **WHEN** a mask such as `*.txt` or `my file*` is given
- **THEN** it appears as one argument and is never expanded by a shell

#### Scenario: Absolute path
- **WHEN** an include or exclude path is absolute, such as `/etc`
- **THEN** building fails with an error saying filter paths must be relative to the root

#### Scenario: Empty filter value
- **WHEN** a mask or path filter is given an empty value
- **THEN** building fails with an error naming the filter

#### Scenario: Filter on isolate
- **WHEN** any filter is combined with an isolate operation
- **THEN** building fails with an error saying filters apply only to create, extract and list

### Requirement: Deterministic argument order
The generated argument list SHALL be deterministic for the same input: the operation first, then the remaining options in a fixed order, with repeated filters in the order they were added and the slice hooks last.

#### Scenario: Repeatable output
- **WHEN** the same command is built twice from the same description
- **THEN** both argument lists are identical

### Requirement: Running the command
The built command SHALL be runnable through a replaceable runner that returns the exit status, standard output and standard error, so callers and tests do not need dar installed.

#### Scenario: Fake runner
- **WHEN** a command is run with a fake runner
- **THEN** the runner receives the program name `dar` and the built arguments and its result is returned to the caller

#### Scenario: Non-zero exit
- **WHEN** dar exits with a non-zero status
- **THEN** the result carries the exit code and captured standard error, and an error is returned

#### Scenario: Live output forwarding
- **WHEN** the default runner is configured with writers for standard output and standard error
- **THEN** dar's output is written to those writers while the command runs, and is still returned in the result

#### Scenario: Custom binary
- **WHEN** a different dar binary path is configured
- **THEN** the runner receives that path instead of `dar`

### Requirement: Slice hooks
The builder SHALL support `-E <command>`, which dar runs when it needs a slice, and `-F <command>`, the same for the reference archive. The command string SHALL be passed to dar as one unmodified argument, including dar's own macros such as `%n`, `%c` and `%p`. Hooks apply to every operation, and an empty command MUST be rejected. Because dar executes the string through a shell, callers are responsible for the content of the command.

#### Scenario: Between-slices hook
- **WHEN** a create command is built with the hook `echo %n >> /tmp/slices.log`
- **THEN** the arguments contain `-E` followed by `echo %n >> /tmp/slices.log` as a single argument

#### Scenario: Reference hook
- **WHEN** an extract command is built with a reference archive and the reference hook `mount-catalog.sh %n`
- **THEN** the arguments contain `-F` followed by `mount-catalog.sh %n` as a single argument

#### Scenario: Empty hook
- **WHEN** a slice hook is given an empty command
- **THEN** building fails with an error naming the hook

### Requirement: Verbose output parsing
The package SHALL provide a parser for the output of dar run with `-v`. The parser SHALL track only the most recent file dar reports working on, for create and extract, and SHALL collect the errors dar reports, in the order they appear. It SHALL accept lines from both standard output and standard error, and ignore lines it does not recognize. The parser does not report slice numbers, because dar's verbose output does not contain them.

#### Scenario: File being created
- **WHEN** the lines `Adding file to archive: /data/a.txt` and `Adding file to archive: /data/b.txt` are parsed in that order
- **THEN** the current file is `/data/b.txt`

#### Scenario: Other entry types
- **WHEN** a line reports a folder, symlink, named pipe or hard link being added, such as `Adding folder to archive: /data/d` or `Recording hard link into the archive: /data/hard`
- **THEN** the current file is the path from that line

#### Scenario: File being restored
- **WHEN** the line `Restoring file's data: /restore/a.txt` is parsed
- **THEN** the current file is `/restore/a.txt`

#### Scenario: Nothing seen yet
- **WHEN** no file line has been parsed
- **THEN** the current file is empty

#### Scenario: Unrelated lines
- **WHEN** lines such as `Building the catalog object...`, `Adding reference to files that have been destroyed since reference backup...`, the summary line ` 0 inode(s) not restored (not saved in archive)` or empty lines are parsed
- **THEN** the current file and the errors are unchanged

#### Scenario: Paths with spaces
- **WHEN** the line `Adding file to archive: /data/my dir/big file.bin` is parsed
- **THEN** the current file is `/data/my dir/big file.bin`

#### Scenario: File error
- **WHEN** the line `Error while saving /data/secret.txt: Permission denied` is parsed
- **THEN** an error is recorded with the path `/data/secret.txt`, the message `Permission denied` and the original line

#### Scenario: Restore error
- **WHEN** the line `Error while restoring /restore/big.bin : Bad CRC, data corruption occurred` is parsed
- **THEN** an error is recorded with the path `/restore/big.bin` and the message `Bad CRC, data corruption occurred`

#### Scenario: File not restored
- **WHEN** a line such as `/restore/big.bin not restored (user choice)` is parsed
- **THEN** an error is recorded with the path `/restore/big.bin` and the reason `user choice`

#### Scenario: Fatal error
- **WHEN** the lines `Aborting due to exception: <message>` and `FATAL error, aborting operation: <message>` are parsed
- **THEN** a single fatal error with that message is recorded, not one per line

#### Scenario: Aborted by prompt
- **WHEN** the line `Aborting program. User refused to continue while asking: <message>` is parsed
- **THEN** a fatal error with that message is recorded, unless one with the same message is already recorded

#### Scenario: Failed hook
- **WHEN** a failing `-E` hook makes dar print `Aborting due to exception: Fatal error on user command line: <detail>` followed by `Aborting program. An error occurred concerning user command execution: Fatal error on user command line: <detail>`
- **THEN** a single fatal error with the message `Fatal error on user command line: <detail>` is recorded

#### Scenario: Latest file survives errors
- **WHEN** an error line is parsed after a file line
- **THEN** the current file is still the last file line

#### Scenario: Output arriving in pieces
- **WHEN** the parser receives output as arbitrary chunks, including chunks that end in the middle of a line, from standard output and standard error concurrently
- **THEN** it assembles complete lines per stream and produces the same result as if each line had been parsed whole, without data races

### Requirement: Slicing listing parsing
The package SHALL provide a parser for the output of `dar -l -Tslicing`. It SHALL read each listed entry with its name, type and the slices that hold its data, ignore header, footer and any other lines that are not entries, and, given a regular expression, return the slices required for the entries whose name matches.

#### Scenario: Single slice
- **WHEN** the line `4<TAB> [Saved][ ]       [-L-][     ][ ] -rw-r--r-- my dir/sub/we ird: name.txt` is parsed
- **THEN** the entry has the name `my dir/sub/we ird: name.txt`, type file and slice 4

#### Scenario: Slice range
- **WHEN** an entry line starts with `1-4`
- **THEN** the entry covers slices 1, 2, 3 and 4

#### Scenario: No slice
- **WHEN** an entry line has an empty slice column, as for symlinks, named pipes and entries without data in this archive
- **THEN** the entry has no slices

#### Scenario: Entry types
- **WHEN** entries for a directory, a symlink and a regular file are parsed
- **THEN** their types are directory, symlink and file

#### Scenario: Non-entry lines
- **WHEN** the header, the `-----` separators and `All displayed files have their data in slice range [1-4]` are parsed
- **THEN** no entry is produced for them and no error is returned

#### Scenario: Removed entries
- **WHEN** a listing of a differential archive contains a line such as `<TAB> [--- REMOVED ENTRY ----]d/sym`
- **THEN** no entry is produced for it and no error is returned

#### Scenario: Needed slices for matching files
- **WHEN** the regular expression `big file` is applied to a listing where `my dir/big file.bin` covers slices 1-4
- **THEN** the needed slices are 1, 2, 3 and 4

#### Scenario: Several matches
- **WHEN** the expression matches one entry in slices 1-2 and another in slices 2-3
- **THEN** the needed slices are 1, 2 and 3, sorted and without duplicates

#### Scenario: Matching against the full name
- **WHEN** the expression is `^docs/.*\.md$`
- **THEN** it is matched against the entry's full name as listed, and only entries whose whole name fits are used

#### Scenario: No match
- **WHEN** no entry name matches
- **THEN** the needed slices are empty and no error is returned

#### Scenario: Matching entry without slices
- **WHEN** the only matching entry has no slices
- **THEN** the needed slices are empty, and the matching entry is still returned to the caller

#### Scenario: Invalid expression
- **WHEN** the regular expression does not compile
- **THEN** an error naming the expression is returned and no slices are computed

#### Scenario: Matched entries returned
- **WHEN** an expression matches several entries
- **THEN** the matched entries are returned in listing order together with the needed slices
