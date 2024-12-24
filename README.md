# PURGE: Python Utilities Rewritten in Go and Enhanced

[简体中文](README.zh-CN.md)

PURGE is a utilities collection where original Python scripts are rewritten in Go and enhanced.

Some original scripts might be overkillers, but this repo generally obeys the UNIX philosophy of "do one thing and do it well" and will take them apart as different standalone utilities.

## Utilities List

- redisuck rewritten from [redis-rogue-server](https://github.com/Dliv3/redis-rogue-server)

### WIP

- Makefile is not yet available
- gitdump from [dumpall](https://github.com/0xHJK/dumpall)
- dsstoredump from [dumpall](https://github.com/0xHJK/dumpall)
- svndump from [dumpall](https://github.com/0xHJK/dumpall)
- [CVE-2021-25646](https://github.com/vulhub/vulhub/blob/master/apache-druid/CVE-2021-25646/README.zh-cn.md)

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
