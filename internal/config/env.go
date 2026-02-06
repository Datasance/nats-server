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
	"os"
	"strconv"
)

const (
	EnvNatsConf            = "NATS_CONF"
	EnvNatsAccounts        = "NATS_ACCOUNTS"
	EnvNatsSSLDir          = "NATS_SSL_DIR"
	EnvNatsServerBin       = "NATS_SERVER_BIN"
	EnvNatsMonitorPort     = "NATS_MONITOR_PORT"
	DefaultNatsConf        = "/etc/nats/config/server.conf"
	DefaultNatsAccounts    = "/etc/nats/config/accounts.conf"
	DefaultNatsSSLDir      = "/etc/nats/certs"
	DefaultNatsServerBin   = "/home/runner/bin/nats-server"
	DefaultNatsMonitorPort = 8222
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
