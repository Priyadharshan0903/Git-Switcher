package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOrEmptyMissingFile(t *testing.T) {
	dir := t.TempDir()
	got, err := ReadOrEmpty(filepath.Join(dir, "does-not-exist"))
	if err != nil {
		t.Fatalf("ReadOrEmpty: %v", err)
	}
	if got != "" {
		t.Fatalf("ReadOrEmpty on missing file = %q, want \"\"", got)
	}
}

func TestReadOrEmptyExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	got, err := ReadOrEmpty(path)
	if err != nil {
		t.Fatalf("ReadOrEmpty: %v", err)
	}
	if got != "hello" {
		t.Fatalf("ReadOrEmpty = %q, want %q", got, "hello")
	}
}
