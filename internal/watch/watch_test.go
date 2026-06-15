package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.conf")
	if FileExists(filePath) {
		t.Fatal("FileExists() = true for missing file")
	}
	if err := os.WriteFile(filePath, []byte("conf"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if !FileExists(filePath) {
		t.Fatal("FileExists() = false for existing file")
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if FileExists(filepath.Join(dir, "subdir")) {
		t.Fatal("FileExists() = true for directory")
	}
}

func TestDir_missingBasePathReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Dir(ctx, filepath.Join(t.TempDir(), "missing"), 10*time.Millisecond, func() {})
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Dir did not return after cancel")
	}
}

func TestConfigFile_missingParentReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	configPath := filepath.Join(t.TempDir(), "missing", "server.conf")
	go func() {
		ConfigFile(ctx, configPath, 10*time.Millisecond, func() {})
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ConfigFile did not return after cancel")
	}
}
