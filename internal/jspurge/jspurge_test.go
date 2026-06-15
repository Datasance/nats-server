package jspurge

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJWT(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("jwt"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestToPurge(t *testing.T) {
	tests := []struct {
		name            string
		accountsWithJS  []string
		currentResolver []string
		want            []string
	}{
		{
			name:            "none to purge when resolver matches",
			accountsWithJS:  []string{"A", "B"},
			currentResolver: []string{"A", "B"},
			want:            nil,
		},
		{
			name:            "purge accounts missing from resolver",
			accountsWithJS:  []string{"A", "B", "C"},
			currentResolver: []string{"A"},
			want:            []string{"B", "C"},
		},
		{
			name:            "all purge when resolver empty",
			accountsWithJS:  []string{"A"},
			currentResolver: nil,
			want:            []string{"A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPurge(tt.accountsWithJS, tt.currentResolver)
			if len(got) != len(tt.want) {
				t.Fatalf("ToPurge() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ToPurge() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestAccountsFromJWTDir(t *testing.T) {
	dir := t.TempDir()
	writeJWT(t, dir, "acct-one.jwt")
	writeJWT(t, dir, "acct-two.jwt")
	writeJWT(t, dir, "skip.jwt.delete")
	writeJWT(t, dir, "not-jwt.txt")

	got, err := AccountsFromJWTDir(dir)
	if err != nil {
		t.Fatalf("AccountsFromJWTDir() err = %v", err)
	}
	want := []string{"acct-one", "acct-two"}
	if len(got) != len(want) {
		t.Fatalf("AccountsFromJWTDir() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("AccountsFromJWTDir() = %v, want %v", got, want)
		}
	}
}

func TestAccountsFromJetStreamStore(t *testing.T) {
	storeDir := t.TempDir()
	jetstreamDir := filepath.Join(storeDir, "jetstream")
	if err := os.MkdirAll(filepath.Join(jetstreamDir, "ACCT1"), 0o755); err != nil {
		t.Fatalf("mkdir ACCT1: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(jetstreamDir, "ACCT2"), 0o755); err != nil {
		t.Fatalf("mkdir ACCT2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jetstreamDir, "file"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := AccountsFromJetStreamStore(storeDir)
	if err != nil {
		t.Fatalf("AccountsFromJetStreamStore() err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("AccountsFromJetStreamStore() = %v, want 2 accounts", got)
	}

	missing, err := AccountsFromJetStreamStore(filepath.Join(storeDir, "missing"))
	if err != nil {
		t.Fatalf("AccountsFromJetStreamStore(missing) err = %v", err)
	}
	if missing != nil {
		t.Fatalf("AccountsFromJetStreamStore(missing) = %v, want nil", missing)
	}
}
