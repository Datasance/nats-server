package config

import (
	"os"
	"path/filepath"
	"testing"
)

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestGetNatsConf_default(t *testing.T) {
	clearEnv(t, EnvNatsConf)
	if got := GetNatsConf(); got != DefaultNatsConf {
		t.Fatalf("GetNatsConf() = %q, want %q", got, DefaultNatsConf)
	}
}

func TestGetNatsConf_override(t *testing.T) {
	t.Setenv(EnvNatsConf, "/custom/server.conf")
	if got := GetNatsConf(); got != "/custom/server.conf" {
		t.Fatalf("GetNatsConf() = %q, want /custom/server.conf", got)
	}
}

func TestGetNatsTLSDir(t *testing.T) {
	tests := []struct {
		name   string
		tlsDir string
		sslDir string
		want   string
	}{
		{
			name: "default when unset",
			want: DefaultNatsTLSDir,
		},
		{
			name:   "NATS_TLS_DIR takes precedence",
			tlsDir: "/tls/primary",
			sslDir: "/ssl/fallback",
			want:   "/tls/primary",
		},
		{
			name:   "NATS_SSL_DIR deprecated fallback",
			sslDir: "/ssl/legacy",
			want:   "/ssl/legacy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, EnvNatsTLSDir, EnvNatsSSLDir)
			if tt.tlsDir != "" {
				t.Setenv(EnvNatsTLSDir, tt.tlsDir)
			}
			if tt.sslDir != "" {
				t.Setenv(EnvNatsSSLDir, tt.sslDir)
			}
			if got := GetNatsTLSDir(); got != tt.want {
				t.Fatalf("GetNatsTLSDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetNatsServerMode(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{name: "default", want: DefaultNatsServerMode},
		{name: "trim and lower", env: "  LEAF  ", want: "leaf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, EnvNatsServerMode)
			if tt.env != "" {
				t.Setenv(EnvNatsServerMode, tt.env)
			}
			if got := GetNatsServerMode(); got != tt.want {
				t.Fatalf("GetNatsServerMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetNatsMonitorPort(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want int
	}{
		{name: "default", want: DefaultNatsMonitorPort},
		{name: "valid override", env: "9090", want: 9090},
		{name: "zero disables", env: "0", want: 0},
		{name: "invalid falls back", env: "not-a-port", want: DefaultNatsMonitorPort},
		{name: "out of range falls back", env: "70000", want: DefaultNatsMonitorPort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, EnvNatsMonitorPort)
			if tt.env != "" {
				t.Setenv(EnvNatsMonitorPort, tt.env)
			}
			if got := GetNatsMonitorPort(); got != tt.want {
				t.Fatalf("GetNatsMonitorPort() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetNatsSysUserCredPath(t *testing.T) {
	clearEnv(t, EnvNatsSysUserCredPath, EnvNatsCredsDir)
	if got := GetNatsSysUserCredPath(); got != "" {
		t.Fatalf("GetNatsSysUserCredPath() = %q, want empty", got)
	}

	t.Setenv(EnvNatsCredsDir, "/etc/nats/creds/")
	t.Setenv(EnvNatsSysUserCredPath, "sys.creds")
	if got := GetNatsSysUserCredPath(); got != filepath.Join("/etc/nats/creds/", "sys.creds") {
		t.Fatalf("GetNatsSysUserCredPath() = %q, want joined relative path", got)
	}

	t.Setenv(EnvNatsSysUserCredPath, "/abs/sys.creds")
	if got := GetNatsSysUserCredPath(); got != "/abs/sys.creds" {
		t.Fatalf("GetNatsSysUserCredPath() = %q, want absolute path", got)
	}
}

func TestGetJetStreamStoreDir(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "server.conf")

	writeConf := func(content string) {
		t.Helper()
		if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}

	t.Run("from env absolute", func(t *testing.T) {
		clearEnv(t, EnvNatsJetStreamStoreDir)
		t.Setenv(EnvNatsJetStreamStoreDir, "/data/jetstream")
		if got := GetJetStreamStoreDir(confPath); got != "/data/jetstream" {
			t.Fatalf("GetJetStreamStoreDir() = %q, want /data/jetstream", got)
		}
	})

	t.Run("from env relative to config dir", func(t *testing.T) {
		clearEnv(t, EnvNatsJetStreamStoreDir)
		t.Setenv(EnvNatsJetStreamStoreDir, "js-store")
		want := filepath.Join(dir, "js-store")
		if got := GetJetStreamStoreDir(confPath); got != want {
			t.Fatalf("GetJetStreamStoreDir() = %q, want %q", got, want)
		}
	})

	t.Run("from config absolute store_dir", func(t *testing.T) {
		clearEnv(t, EnvNatsJetStreamStoreDir)
		writeConf("jetstream {\n  store_dir: \"/var/lib/jetstream\"\n}\n")
		if got := GetJetStreamStoreDir(confPath); got != "/var/lib/jetstream" {
			t.Fatalf("GetJetStreamStoreDir() = %q, want /var/lib/jetstream", got)
		}
	})

	t.Run("from config relative store_dir", func(t *testing.T) {
		clearEnv(t, EnvNatsJetStreamStoreDir)
		writeConf("jetstream {\n  store_dir: \"data/js\"\n}\n")
		want := filepath.Join(dir, "data/js")
		if got := GetJetStreamStoreDir(confPath); got != want {
			t.Fatalf("GetJetStreamStoreDir() = %q, want %q", got, want)
		}
	})

	t.Run("missing config returns empty", func(t *testing.T) {
		clearEnv(t, EnvNatsJetStreamStoreDir)
		missing := filepath.Join(dir, "missing.conf")
		if got := GetJetStreamStoreDir(missing); got != "" {
			t.Fatalf("GetJetStreamStoreDir() = %q, want empty", got)
		}
	})
}
