# PURGE: Python Utilities Rewritten in Go and Enhanced

[简体中文](README.zh-CN.md)

PURGE is a collection of Python utilities that have been rewritten in Go and enhanced.

Some utilities might be overkillers, but this repo generally obeys the UNIX philosophy of "do one thing and do it well" and will take them apart as different standalone utilities.

## Utilities List

- redisuck rewritten from [redis-rogue-server](https://github.com/Dliv3/redis-rogue-server)

### WIP

- Makefile is not yet available
- gitdump from [dumpall](https://github.com/0xHJK/dumpall)
- dsstoredump from [dumpall](https://github.com/0xHJK/dumpall)
- svndump from [dumpall](https://github.com/0xHJK/dumpall)

## Build

### Windows

Prerequisites: Go 1.22+

```powershell
./make.cmd
```

### Linux & macOS

Prerequisites: Go 1.22+ and Make

```bash
make
```
