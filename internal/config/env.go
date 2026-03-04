/*
 *  *******************************************************************************
 *  * Copyright (c) 2023 Datasance Teknoloji A.S.
 *  *
 *  * This program and the accompanying materials are made available under the
 *  * terms of the Eclipse Public License v. 2.0 which is available at
 *  * http://www.eclipse.org/legal/epl-2.0
 *  *
 *  * SPDX-License-Identifier: EPL-2.0
 *  *******************************************************************************
 */

package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	EnvNatsConf               = "NATS_CONF"
	EnvNatsAccounts           = "NATS_ACCOUNTS"
	EnvNatsSSLDir             = "NATS_SSL_DIR"
	EnvNatsJWTDir             = "NATS_JWT_DIR"
	EnvNatsJWTMountDir        = "NATS_JWT_MOUNT_DIR"
	EnvNatsServerMode         = "NATS_SERVER_MODE"
	EnvNatsCredsDir           = "NATS_CREDS_DIR"
	EnvNatsServerBin          = "NATS_SERVER_BIN"
	EnvNatsMonitorPort        = "NATS_MONITOR_PORT"
	EnvNatsSysUserCredPath    = "NATS_SYS_USER_CRED_PATH"
	EnvNatsClientURL          = "NATS_CLIENT_URL"
	EnvNatsJetStreamStoreDir  = "NATS_JETSTREAM_STORE_DIR"
	DefaultNatsConf           = "/etc/nats/config/server.conf"
	DefaultNatsAccounts       = "/etc/nats/config/accounts.conf"
	DefaultNatsSSLDir         = "/etc/nats/certs"
	DefaultNatsJWTDir         = "/home/runner/nats/jwt"
	DefaultNatsJWTMountDir    = "/tmp/nats/jwt"
	DefaultNatsServerMode     = "server"
	DefaultNatsCredsDir       = "/etc/nats/creds/"
	DefaultNatsServerBin      = "/home/runner/bin/nats-server"
	DefaultNatsMonitorPort    = 8222
	DefaultNatsClientURL      = "nats://127.0.0.1:4222"
)

// GetNatsConf returns the server config file path from NATS_CONF, or DefaultNatsConf if unset.
func GetNatsConf() string {
	if p := os.Getenv(EnvNatsConf); p != "" {
		return p
	}
	return DefaultNatsConf
}

// GetNatsAccounts returns the account config file path from NATS_ACCOUNTS, or DefaultNatsAccounts if unset.
func GetNatsAccounts() string {
	if p := os.Getenv(EnvNatsAccounts); p != "" {
		return p
	}
	return DefaultNatsAccounts
}

// GetNatsSSLDir returns the SSL certs directory from NATS_SSL_DIR, or DefaultNatsSSLDir if unset.
func GetNatsSSLDir() string {
	if p := os.Getenv(EnvNatsSSLDir); p != "" {
		return p
	}
	return DefaultNatsSSLDir
}

// GetNatsJWTDir returns the JWT directory from NATS_JWT_DIR, or DefaultNatsJWTDir if unset.
func GetNatsJWTDir() string {
	if p := os.Getenv(EnvNatsJWTDir); p != "" {
		return p
	}
	return DefaultNatsJWTDir
}

// GetNatsJWTMountDir returns the JWT mount directory from NATS_JWT_MOUNT_DIR, or DefaultNatsJWTMountDir if unset.
func GetNatsJWTMountDir() string {
	if p := os.Getenv(EnvNatsJWTMountDir); p != "" {
		return p
	}
	return DefaultNatsJWTMountDir
}

// GetNatsServerMode returns the server mode from NATS_SERVER_MODE, or DefaultNatsServerMode if unset.
// Returned value is trimmed and lowercased for comparison (e.g. "server", "leaf").
func GetNatsServerMode() string {
	s := strings.TrimSpace(os.Getenv(EnvNatsServerMode))
	if s == "" {
		return DefaultNatsServerMode
	}
	return strings.ToLower(s)
}

// GetNatsCredsDir returns the creds directory from NATS_CREDS_DIR, or DefaultNatsCredsDir if unset.
func GetNatsCredsDir() string {
	if p := os.Getenv(EnvNatsCredsDir); p != "" {
		return p
	}
	return DefaultNatsCredsDir
}

// GetNatsServerBin returns the nats-server binary path from NATS_SERVER_BIN, or DefaultNatsServerBin if unset.
func GetNatsServerBin() string {
	if p := os.Getenv(EnvNatsServerBin); p != "" {
		return p
	}
	return DefaultNatsServerBin
}

// GetNatsMonitorPort returns the HTTP monitoring port from NATS_MONITOR_PORT, or DefaultNatsMonitorPort (8222) if unset or invalid.
// Set to 0 to disable monitoring.
func GetNatsMonitorPort() int {
	s := os.Getenv(EnvNatsMonitorPort)
	if s == "" {
		return DefaultNatsMonitorPort
	}
	port, err := strconv.Atoi(s)
	if err != nil || port < 0 || port > 65535 {
		return DefaultNatsMonitorPort
	}
	return port
}

// GetNatsSysUserCredPath returns the system user credentials file path from NATS_SYS_USER_CRED_PATH.
// If the value is not an absolute path, it is resolved relative to NATS_CREDS_DIR.
// Returns empty string if unset.
func GetNatsSysUserCredPath() string {
	p := os.Getenv(EnvNatsSysUserCredPath)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(GetNatsCredsDir(), p)
}

// GetNatsClientURL returns the NATS client URL from NATS_CLIENT_URL, or DefaultNatsClientURL (nats://127.0.0.1:4222) if unset.
func GetNatsClientURL() string {
	if u := os.Getenv(EnvNatsClientURL); u != "" {
		return u
	}
	return DefaultNatsClientURL
}

// GetJetStreamStoreDir returns the JetStream store directory. If NATS_JETSTREAM_STORE_DIR is set, uses it
// (resolving relative paths against the server config file's directory). Otherwise parses jetstream.store_dir
// from the server config file. Returns empty string if unset or parse fails.
func GetJetStreamStoreDir(serverConfPath string) string {
	if p := os.Getenv(EnvNatsJetStreamStoreDir); p != "" {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(filepath.Dir(serverConfPath), p)
	}
	storeDir := parseJetStreamStoreDirFromConfig(serverConfPath)
	if storeDir == "" {
		return ""
	}
	if filepath.IsAbs(storeDir) {
		return storeDir
	}
	return filepath.Join(filepath.Dir(serverConfPath), storeDir)
}

// parseJetStreamStoreDirFromConfig reads the server config file and extracts jetstream.store_dir value.
// Returns empty string on any error or if not found.
func parseJetStreamStoreDirFromConfig(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	var inJetstream bool
	var braceCount int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "jetstream") && strings.Contains(line, "{") {
			inJetstream = true
			braceCount = 1
			continue
		}
		if inJetstream {
			if strings.Contains(line, "{") {
				braceCount++
			}
			if strings.Contains(line, "}") {
				braceCount--
				if braceCount == 0 {
					inJetstream = false
				}
			}
			if strings.HasPrefix(line, "store_dir") {
				idx := strings.Index(line, ":")
				if idx == -1 {
					continue
				}
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, `"`)
				if val != "" {
					return val
				}
			}
		}
	}
	return ""
}
