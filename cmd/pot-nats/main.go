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
	"log"
	"os"
	"sync"
	"time"

	"github.com/datasance/nats-server/internal/claimspush"
	"github.com/datasance/nats-server/internal/config"
	"github.com/datasance/nats-server/internal/jspurge"
	"github.com/datasance/nats-server/internal/nats"
	"github.com/datasance/nats-server/internal/watch"
)

const (
	configWaitAttempts    = 30
	configWaitInterval    = time.Second
	reconcileStartDelay   = 3 * time.Second
	reconcileAfterReload  = 3 * time.Second
)

func main() {
	natsConf := config.GetNatsConf()
	natsAccounts := config.GetNatsAccounts()
	natsSSLDir := config.GetNatsSSLDir()
	natsJWTDir := config.GetNatsJWTDir()
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

	server := new(nats.Server)
	exitCh := make(chan error, 1)

	if err := server.Start(natsConf, exitCh); err != nil {
		log.Fatalf("Failed to start NATS server: %v", err)
	}

	// One-time JetStream account reconciliation after startup (e.g. purge accounts removed while process was down).
	go func() {
		time.Sleep(reconcileStartDelay)
		runJetStreamReconcile(natsConf, natsJWTDir)
	}()

	ctx := context.Background()
	debounce := 500 * time.Millisecond

	// Coalescer: multiple watchers report a cause; one debounced reload runs, with reconcile+claims push only when jwt was a cause.
	var (
		coalescerMu    sync.Mutex
		coalescerCauses map[string]bool
		coalescerTimer *time.Timer
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
			if err := server.Reload(); err != nil {
				log.Printf("Reload after change: %v", err)
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

	// Watch JWT directory if it exists; on change coalesced reload, then reconcile and claims push when jwt is a cause
	if info, err := os.Stat(natsJWTDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsJWTDir, debounce, func() { scheduleReload("jwt") })
	}

	// Watch creds directory if it exists
	if info, err := os.Stat(natsCredsDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsCredsDir, debounce, func() { scheduleReload("creds") })
	}

	err := <-exitCh
	if err != nil {
		log.Printf("NATS server exited: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
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
