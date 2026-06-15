package jwtcopy

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJWT(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestSyncMountToJWT_sameDirSkips(t *testing.T) {
	dir := t.TempDir()
	copied, removed, err := SyncMountToJWT(dir, dir)
	if err != nil {
		t.Fatalf("SyncMountToJWT() err = %v", err)
	}
	if copied != 0 || removed != 0 {
		t.Fatalf("SyncMountToJWT() = copied %d removed %d, want 0, 0", copied, removed)
	}
}

func TestSyncMountToJWT_missingMountDir(t *testing.T) {
	jwtDir := t.TempDir()
	copied, removed, err := SyncMountToJWT(filepath.Join(jwtDir, "missing"), jwtDir)
	if err != nil {
		t.Fatalf("SyncMountToJWT() err = %v", err)
	}
	if copied != 0 || removed != 0 {
		t.Fatalf("SyncMountToJWT() = copied %d removed %d, want 0, 0", copied, removed)
	}
}

func TestSyncMountToJWT_emptyMountSkipsRemoval(t *testing.T) {
	mountDir := t.TempDir()
	jwtDir := t.TempDir()
	writeJWT(t, jwtDir, "orphan.jwt", "old")

	copied, removed, err := SyncMountToJWT(mountDir, jwtDir)
	if err != nil {
		t.Fatalf("SyncMountToJWT() err = %v", err)
	}
	if copied != 0 || removed != 0 {
		t.Fatalf("SyncMountToJWT() = copied %d removed %d, want 0, 0", copied, removed)
	}
	if _, err := os.Stat(filepath.Join(jwtDir, "orphan.jwt")); err != nil {
		t.Fatalf("orphan.jwt should remain when mount is empty: %v", err)
	}
}

func TestSyncMountToJWT_copyAndRemoveOrphans(t *testing.T) {
	mountDir := t.TempDir()
	jwtDir := t.TempDir()

	writeJWT(t, mountDir, "acct-a.jwt", "jwt-a")
	writeJWT(t, mountDir, "acct-b.jwt", "jwt-b")
	writeJWT(t, jwtDir, "acct-a.jwt", "stale-a")
	writeJWT(t, jwtDir, "orphan.jwt", "remove-me")
	writeJWT(t, jwtDir, "marked.jwt.delete", "skip")

	copied, removed, err := SyncMountToJWT(mountDir, jwtDir)
	if err != nil {
		t.Fatalf("SyncMountToJWT() err = %v", err)
	}
	if copied != 2 {
		t.Fatalf("copied = %d, want 2", copied)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}

	gotA, err := os.ReadFile(filepath.Join(jwtDir, "acct-a.jwt"))
	if err != nil {
		t.Fatalf("read acct-a.jwt: %v", err)
	}
	if string(gotA) != "jwt-a" {
		t.Fatalf("acct-a.jwt = %q, want jwt-a", string(gotA))
	}

	if _, err := os.Stat(filepath.Join(jwtDir, "orphan.jwt")); !os.IsNotExist(err) {
		t.Fatalf("orphan.jwt should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(jwtDir, "marked.jwt.delete")); err != nil {
		t.Fatalf("marked.jwt.delete should remain: %v", err)
	}
}

func TestListJWTFileNames(t *testing.T) {
	dir := t.TempDir()
	writeJWT(t, dir, "one.jwt", "1")
	writeJWT(t, dir, "two.jwt", "2")
	writeJWT(t, dir, "skip.jwt.delete", "x")
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	names, err := listJWTFileNames(dir)
	if err != nil {
		t.Fatalf("listJWTFileNames() err = %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("listJWTFileNames() = %v, want 2 entries", names)
	}
}
