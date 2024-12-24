package main_test

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/chengongpp/purge/gitdump/gin"
)

func TestGitIndexParser(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	fd, err := os.Open("../.git/index")
	if err != nil {
		t.Fatalf("Failed to open index file due to error: %v", err)
	}
	idxContent, err := io.ReadAll(fd)
	if err != nil {
		t.Fatalf("Failed to read index file due to error: %v", err)
	}
	symbols := gin.ParseIndexContent([]byte(idxContent))
	for s := range symbols {
		switch s.(type) {
		case gin.Entry:
			entry := s.(gin.Entry)
			t.Logf("Entry: %v", entry)
		case gin.Extension:
			extension := s.(gin.Extension)
			t.Logf("Extension: %v", extension)
		case gin.Checksum:
			checksum := s.(gin.Checksum)
			t.Logf("Checksum: %v", checksum)
		}
	}
}

func TestWtf(t *testing.T) {
	fmt.Println(0 % 8)
}
