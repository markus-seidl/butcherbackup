# dar-command-builder Specification

## Purpose
Defines how butcherbackup describes a dar operation and turns it into a validated dar command line, so callers cannot produce invalid option combinations.

## Requirements

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
The builder SHALL support these options and emit the matching dar arguments: root directory `-R <path>`, reference archive `-A <archive>`, on-the-fly isolated catalogue `-@ <archive>`, compression `-z`, slice size `-s <size>`, first slice size `-S <size>`, verbose `-v`, first-slice header `-affs`, list format `-T<format>` and the file filters `-I`, `-X`, `-g` and `-P`.

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
The builder SHALL reject options that do not apply to the chosen operation: isolate requires a reference archive, on-the-fly isolation applies only to create, and the root directory applies only to create and extract.

#### Scenario: Isolate without reference
- **WHEN** an isolate operation is built without a reference archive
- **THEN** building fails with an error saying isolate requires a reference archive

#### Scenario: On-the-fly isolation on extract
- **WHEN** an on-the-fly isolated catalogue is combined with an extract operation
- **THEN** building fails with an error saying it applies only to create

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
The generated argument list SHALL be deterministic for the same input: the operation first, then the remaining options in a fixed order, with repeated filters in the order they were added.

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

#### Scenario: Custom binary
- **WHEN** a different dar binary path is configured
- **THEN** the runner receives that path instead of `dar`
