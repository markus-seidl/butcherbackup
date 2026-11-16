
# About

This is butcher backup, so called since it splits the files with simple shell commands and stores them to HDDs or Tapes via cp or mbuffer commands.



# Usage

```
butcherbackup <command> [arguments]
```

Run `butcherbackup help` to list commands and `butcherbackup <command> -h` for a command's arguments.

```
butcherbackup backup --source /data --destination /mnt/disk1 [--exclude <pattern>]...
butcherbackup restore --backup /mnt/disk1 --target /restore [file]...
```

`backup` and `restore` are skeletons for now: they parse their arguments and report that they are not implemented yet.
