package logger

import (
	"os"
	"testing"
)

func TestRotatingFileWriterKeepsOnlyCurrentFile(t *testing.T) {
	dir := t.TempDir()
	writer, err := newRotatingFileWriter(dir)
	if err != nil {
		t.Fatalf("create rotating writer: %v", err)
	}

	chunk := make([]byte, 1024*1024)
	for i := 0; i < 101; i++ {
		if _, err := writer.Write(chunk); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close rotating writer: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read log directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one current log file, got %d", len(entries))
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("stat current log: %v", err)
	}
	if info.Size() > maxLogFileSize {
		t.Fatalf("log file exceeded limit: %d", info.Size())
	}
}
