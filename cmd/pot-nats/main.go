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

package main

import (
	"context"
	"crypto/sha256"
	"log"
	"os"
	"sync"
	"time"

	"github.com/datasance/nats-server/internal/claimspush"
	"github.com/datasance/nats-server/internal/config"
	"github.com/datasance/nats-server/internal/jspurge"
	"github.com/datasance/nats-server/internal/jwtcopy"
	"github.com/datasance/nats-server/internal/nats"
	"github.com/datasance/nats-server/internal/watch"
)

const (
	configWaitAttempts   = 30
	configWaitInterval   = time.Second
	reconcileStartDelay  = 3 * time.Second
	reconcileAfterReload = 3 * time.Second
)

func main() {
	natsConf := config.GetNatsConf()
	natsAccounts := config.GetNatsAccounts()
	natsSSLDir := config.GetNatsSSLDir()
	natsJWTDir := config.GetNatsJWTDir()
	natsJWTMountDir := config.GetNatsJWTMountDir()
	natsCredsDir := config.GetNatsCredsDir()

	// Wait for server config file to exist (e.g. volume-mounted by K8s or Pot agent)
	for i := 0; i < configWaitAttempts; i++ {
		if watch.FileExists(natsConf) {
			break
		}
		if i == configWaitAttempts-1 {
			log.Fatalf("NATS config file not found at %s after %d attempts", natsConf, configWaitAttempts)
		}
		log.Printf("Waiting for NATS config at %s...", natsConf)
		time.Sleep(configWaitInterval)
	}

	// Store initial config file hash so we only reload/restart when content actually changes (e.g. avoid ConfigMap re-render with same content).
	var (
		configHashMu   sync.Mutex
		lastConfigHash [32]byte
	)
	if h, err := fileSHA256(natsConf); err == nil {
		lastConfigHash = h
	}

	// Serialize JWT sync so startup and watcher never run SyncMountToJWT concurrently.
	var jwtSyncMu sync.Mutex
	// Sync JWT mount dir to JWT dir before starting nats-server (so writable dir is populated).
	if info, err := os.Stat(natsJWTMountDir); err == nil && info.IsDir() {
		jwtSyncMu.Lock()
		copied, removed, err := jwtcopy.SyncMountToJWT(natsJWTMountDir, natsJWTDir)
		jwtSyncMu.Unlock()
		if err != nil {
			log.Printf("JWT sync at startup failed: %v", err)
		} else {
			log.Printf("JWT sync at startup: copied=%d removed=%d (mount=%s -> jwt=%s)", copied, removed, natsJWTMountDir, natsJWTDir)
		}
	}

	server := new(nats.Server)
	exitCh := make(chan error, 1)
	var (
		restartMu        sync.Mutex
		restartRequested bool
	)

	startServer := func() {
		if err := server.Start(natsConf, exitCh); err != nil {
			log.Fatalf("Failed to start NATS server: %v", err)
		}
	}
	startServer()

	// One-time JetStream account reconciliation after startup (e.g. purge accounts removed while process was down).
	go func() {
		time.Sleep(reconcileStartDelay)
		runJetStreamReconcile(natsConf, natsJWTDir)
	}()

	ctx := context.Background()
	debounce := 500 * time.Millisecond

	// Coalescer: multiple watchers report a cause; one debounced reload runs, with reconcile+claims push only when jwt was a cause.
	var (
		coalescerMu     sync.Mutex
		coalescerCauses map[string]bool
		coalescerTimer  *time.Timer
	)
	scheduleReload := func(cause string) {
		coalescerMu.Lock()
		defer coalescerMu.Unlock()
		if coalescerCauses == nil {
			coalescerCauses = make(map[string]bool)
		}
		coalescerCauses[cause] = true
		if coalescerTimer != nil {
			coalescerTimer.Stop()
		}
			coalescerTimer = time.AfterFunc(debounce, func() {
			coalescerMu.Lock()
			causes := coalescerCauses
			coalescerCauses = nil
			coalescerTimer = nil
			coalescerMu.Unlock()
			// Only treat config as changed if file content actually changed (avoids unnecessary reload/restart on ConfigMap re-render).
			if causes["config"] {
				if h, err := fileSHA256(natsConf); err != nil {
					// Read failed; assume changed so we still react
				} else {
					configHashMu.Lock()
					if h == lastConfigHash {
						causes["config"] = false
					} else {
						lastConfigHash = h
					}
					configHashMu.Unlock()
				}
			}
			if causes["jwt"] {
				jwtSyncMu.Lock()
				copied, removed, err := jwtcopy.SyncMountToJWT(natsJWTMountDir, natsJWTDir)
				jwtSyncMu.Unlock()
				if err != nil {
					log.Printf("JWT sync after mount dir change failed: %v", err)
				} else {
					log.Printf("JWT sync after change: copied=%d removed=%d", copied, removed)
				}
			}
			// Leaf supports reload only for SSL/TLS cert changes; server supports full reload.
			// For leaf with non-SSL changes (config, accounts, jwt, creds), SIGINT and restart so new config is loaded.
			if config.GetNatsServerMode() != "leaf" || causes["ssl"] {
				if err := server.Reload(); err != nil {
					log.Printf("Reload after change: %v", err)
				}
			} else if config.GetNatsServerMode() == "leaf" && (causes["config"] || causes["accounts"] || causes["creds"]) {
				restartMu.Lock()
				restartRequested = true
				restartMu.Unlock()
				if err := server.Stop(); err != nil {
					log.Printf("Stop for restart after change: %v", err)
				}
			}
			if causes["jwt"] {
				go func() {
					time.Sleep(reconcileAfterReload)
					runJetStreamReconcile(natsConf, natsJWTDir)
					credsPath := config.GetNatsSysUserCredPath()
					clientURL := config.GetNatsClientURL()
					claimspush.PushAccountJWTs(ctx, natsJWTDir, clientURL, credsPath, 10*time.Second)
				}()
			}
		})
	}

	// Watch server config file; on change trigger coalesced reload
	go watch.WatchConfigFile(ctx, natsConf, debounce, func() { scheduleReload("config") })

	// Watch account config file if it exists
	if watch.FileExists(natsAccounts) {
		go watch.WatchConfigFile(ctx, natsAccounts, debounce, func() { scheduleReload("accounts") })
	}

	// Watch SSL directory if it exists
	if info, err := os.Stat(natsSSLDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsSSLDir, debounce, func() { scheduleReload("ssl") })
	}

	// Watch JWT mount directory if it exists; on change sync to JWT dir, coalesced reload/restart, then reconcile and claims push
	if info, err := os.Stat(natsJWTMountDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsJWTMountDir, debounce, func() { scheduleReload("jwt") })
	}

	// Watch creds directory if it exists
	if info, err := os.Stat(natsCredsDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsCredsDir, debounce, func() { scheduleReload("creds") })
	}

	for {
		err := <-exitCh
		restartMu.Lock()
		r := restartRequested
		if r {
			restartRequested = false
		}
		restartMu.Unlock()
		if r {
			log.Printf("NATS server stopped for restart, starting again")
			startServer()
			continue
		}
		if err != nil {
			log.Printf("NATS server exited: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

// fileSHA256 returns the SHA256 hash of the file at path, or an error if the file cannot be read.
func fileSHA256(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}

// runJetStreamReconcile computes accounts with JetStream data but not in the resolver, then purges each via the JetStream Account Purge API.
// Logs reconciliation start/skip and per-account purge result, inline with existing log style.
func runJetStreamReconcile(serverConfPath, jwtDir string) {
	storeDir := config.GetJetStreamStoreDir(serverConfPath)
	if storeDir == "" {
		log.Printf("ERROR: JetStream store dir not set or unreadable, skipping account purge reconciliation")
		return
	}
	accountsWithJS, err := jspurge.AccountsFromJetStreamStore(storeDir)
	if err != nil {
		log.Printf("ERROR: JetStream account reconciliation failed to list store: %v", err)
		return
	}
	currentResolver, err := jspurge.AccountsFromJWTDir(jwtDir)
	if err != nil {
		log.Printf("ERROR: JetStream account reconciliation failed to list JWT dir: %v", err)
		return
	}
	toPurge := jspurge.ToPurge(accountsWithJS, currentResolver)
	credsPath := config.GetNatsSysUserCredPath()
	clientURL := config.GetNatsClientURL()

	log.Printf("JetStream account reconciliation: store_dir=%s, resolver_accounts=%d, to_purge=%d", storeDir, len(currentResolver), len(toPurge))
	if credsPath == "" {
		log.Printf("NATS_SYS_USER_CRED_PATH unset, skipping purge API calls")
		return
	}
	ctx := context.Background()
	for _, account := range toPurge {
		if err := jspurge.PurgeAccount(ctx, clientURL, credsPath, account); err != nil {
			log.Printf("ERROR: JetStream account purge failed for %s: %v", account, err)
			continue
		}
		log.Printf("JetStream account purge initiated for %s", account)
	}
}
