package gin

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
)

type Index struct {
	Signature string
	Version   uint32
	Entries   uint32
}

type Entry struct {
	Entry        int
	Ctime        uint64
	Mtime        uint64
	Dev          uint32
	Ino          uint32
	Mode         string
	Uid          uint32
	Gid          uint32
	Size         uint32
	Sha1         string
	Flags        uint16
	AssumeValid  bool
	Extended     bool
	Stage        [2]bool
	Name         string
	ExtraFlags   uint16
	Reserved     bool
	SkipWorktree bool
	IntentToAdd  bool
}

type Extension struct {
	Extension int
	Signature string
	Size      uint32
	Data      string
}

type Checksum struct {
	Checksum bool
	Sha1     string
}

func ParseIndexContent(content []byte) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)

		reader := bytes.NewReader(content)

		var index Index
		signature := make([]byte, 4)
		_, err := reader.Read(signature)
		if err != nil {
			slog.Debug("Failed to read signature", "error", err)
			fmt.Fprintln(os.Stderr, "error:", "Failed to read signature")
			os.Exit(1)
		}
		index.Signature = string(signature)
		if index.Signature != "DIRC" {
			slog.Debug("Not a Git index file", "signature", index.Signature)
			fmt.Fprintln(os.Stderr, "error:", "Not a Git index file")
			os.Exit(1)
		}

		err = binary.Read(reader, binary.BigEndian, &index.Version)
		if err != nil {
			slog.Debug("Failed to read version", "error", err)
			fmt.Fprintln(os.Stderr, "error:", "Failed to read version")
			os.Exit(1)
		}
		if index.Version != 2 && index.Version != 3 {
			slog.Debug("Unsupported version", "version", index.Version)
			fmt.Fprintln(os.Stderr, "error:", fmt.Sprintf("Unsupported version: %d", index.Version))
			os.Exit(1)
		}

		err = binary.Read(reader, binary.BigEndian, &index.Entries)
		if err != nil {
			slog.Debug("Failed to read entries", "error", err)
			fmt.Fprintln(os.Stderr, "error:", "Failed to read entries")
			os.Exit(1)
		}
		out <- index

		for n := uint32(0); n < index.Entries; n++ {
			slog.Debug("Reading entry", "entry", n+1, "total", index.Entries)
			var entry Entry
			entry.Entry = int(n + 1)

			var ctimeSeconds, ctimeNanoseconds, mtimeSeconds, mtimeNanoseconds uint32
			err = binary.Read(reader, binary.BigEndian, &ctimeSeconds)
			if err != nil {
				slog.Debug("Failed to read ctimeSeconds", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read ctimeSeconds")
				os.Exit(1)
			}
			err = binary.Read(reader, binary.BigEndian, &ctimeNanoseconds)
			if err != nil {
				slog.Debug("Failed to read ctimeNanoseconds", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read ctimeNanoseconds")
				os.Exit(1)
			}
			entry.Ctime = uint64(ctimeSeconds)<<32 | uint64(ctimeNanoseconds)

			err = binary.Read(reader, binary.BigEndian, &mtimeSeconds)
			if err != nil {
				slog.Debug("Failed to read mtimeSeconds", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read mtimeSeconds")
				os.Exit(1)
			}
			err = binary.Read(reader, binary.BigEndian, &mtimeNanoseconds)
			if err != nil {
				slog.Debug("Failed to read mtimeNanoseconds", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read mtimeNanoseconds")
				os.Exit(1)
			}
			entry.Mtime = uint64(mtimeSeconds)<<32 | uint64(mtimeNanoseconds)

			err = binary.Read(reader, binary.BigEndian, &entry.Dev)
			if err != nil {
				slog.Debug("Failed to read dev", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read dev")
				os.Exit(1)
			}
			err = binary.Read(reader, binary.BigEndian, &entry.Ino)
			if err != nil {
				slog.Debug("Failed to read ino", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read ino")
				os.Exit(1)
			}
			var mode uint32
			err = binary.Read(reader, binary.BigEndian, &mode)
			if err != nil {
				slog.Debug("Failed to read mode", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read mode")
				os.Exit(1)
			}
			entry.Mode = fmt.Sprintf("%06o", mode)
			err = binary.Read(reader, binary.BigEndian, &entry.Uid)
			if err != nil {
				slog.Debug("Failed to read uid", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read uid")
				os.Exit(1)
			}
			err = binary.Read(reader, binary.BigEndian, &entry.Gid)
			if err != nil {
				slog.Debug("Failed to read gid", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read gid")
				os.Exit(1)
			}
			err = binary.Read(reader, binary.BigEndian, &entry.Size)
			if err != nil {
				slog.Debug("Failed to read size", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read size")
				os.Exit(1)
			}

			sha1 := make([]byte, 20)
			_, err = reader.Read(sha1)
			if err != nil {
				slog.Debug("Failed to read sha1", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read sha1")
				os.Exit(1)
			}
			entry.Sha1 = hex.EncodeToString(sha1)
			err = binary.Read(reader, binary.BigEndian, &entry.Flags)
			if err != nil {
				slog.Debug("Failed to read flags", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read flags")
				os.Exit(1)
			}

			entry.AssumeValid = entry.Flags&(0b10000000<<8) != 0
			entry.Extended = entry.Flags&(0b01000000<<8) != 0
			entry.Stage[0] = entry.Flags&(0b00100000<<8) != 0
			entry.Stage[1] = entry.Flags&(0b00010000<<8) != 0

			entryLen := 62

			if entry.Extended && index.Version == 3 {
				err = binary.Read(reader, binary.BigEndian, &entry.ExtraFlags)
				if err != nil {
					slog.Debug("Failed to read extra flags", "error", err)
					fmt.Fprintln(os.Stderr, "error:", "Failed to read extra flags")
					os.Exit(1)
				}
				entry.Reserved = entry.ExtraFlags&(0b10000000<<8) != 0
				entry.SkipWorktree = entry.ExtraFlags&(0b01000000<<8) != 0
				entry.IntentToAdd = entry.ExtraFlags&(0b00100000<<8) != 0
				entryLen += 2
			}

			namelen := entry.Flags & 0x0FFF
			if namelen < 0x0FFF {
				name := make([]byte, namelen)
				_, err = reader.Read(name)
				if err != nil {
					slog.Debug("Failed to read name", "error", err)
					fmt.Fprintln(os.Stderr, "error:", "Failed to read name")
					os.Exit(1)
				}
				entry.Name = string(name)
				entryLen += int(namelen)
			} else {
				var name []byte
				for {
					b := make([]byte, 1)
					_, err = reader.Read(b)
					if err != nil {
						slog.Debug("Failed to read name byte", "error", err)
						fmt.Fprintln(os.Stderr, "error:", "Failed to read name byte")
						os.Exit(1)
					}
					if b[0] == 0 {
						break
					}
					name = append(name, b[0])
				}
				entry.Name = string(name)
				entryLen += 1
			}
			padLen := (8 - (entryLen % 8))
			slog.Debug("Length Calc", "entryLen", entryLen, "padLen", padLen)
			padding := make([]byte, padLen)
			slog.Debug("Entry", "entry", entry)
			_, err = reader.Read(padding)
			if err != nil {
				slog.Debug("Failed to read padding", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read padding")
				os.Exit(1)
			}
			out <- entry
		}

		for reader.Len() > 20 {
			var extension Extension
			signature := make([]byte, 4)
			_, err := reader.Read(signature)
			if err != nil {
				slog.Debug("Failed to read extension signature", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read extension signature")
				os.Exit(1)
			}
			extension.Signature = string(signature)
			err = binary.Read(reader, binary.BigEndian, &extension.Size)
			if err != nil {
				slog.Debug("Failed to read extension size", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read extension size")
				os.Exit(1)
			}
			data := make([]byte, extension.Size)
			_, err = reader.Read(data)
			if err != nil {
				slog.Debug("Failed to read extension data", "error", err)
				fmt.Fprintln(os.Stderr, "error:", "Failed to read extension data")
				os.Exit(1)
			}
			extension.Data = string(data)
			out <- extension
		}

		var checksum Checksum
		checksum.Checksum = true
		sha1 := make([]byte, 20)
		_, err = reader.Read(sha1)
		if err != nil {
			slog.Debug("Failed to read checksum sha1", "error", err)
			fmt.Fprintln(os.Stderr, "error:", "Failed to read checksum sha1")
			os.Exit(1)
		}
		checksum.Sha1 = hex.EncodeToString(sha1)
		out <- checksum
	}()
	return out
}
