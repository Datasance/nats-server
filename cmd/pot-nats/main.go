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
	"time"

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

	// Watch server config file; on change trigger reload (SIGHUP)
	go watch.WatchConfigFile(ctx, natsConf, debounce, func() {
		if err := server.Reload(); err != nil {
			log.Printf("Reload after config change: %v", err)
		}
	})

	// Watch account config file if it exists
	if watch.FileExists(natsAccounts) {
		go watch.WatchConfigFile(ctx, natsAccounts, debounce, func() {
			if err := server.Reload(); err != nil {
				log.Printf("Reload after accounts change: %v", err)
			}
		})
	}

	// Watch SSL directory if it exists
	if info, err := os.Stat(natsSSLDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsSSLDir, debounce, func() {
			if err := server.Reload(); err != nil {
				log.Printf("Reload after SSL dir change: %v", err)
			}
		})
	}

	// Watch JWT directory if it exists; on change reload and reconcile JetStream (purge removed accounts).
	if info, err := os.Stat(natsJWTDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsJWTDir, debounce, func() {
			if err := server.Reload(); err != nil {
				log.Printf("Reload after JWT dir change: %v", err)
			}
			go func() {
				time.Sleep(reconcileAfterReload)
				runJetStreamReconcile(natsConf, natsJWTDir)
			}()
		})
	}

	// Watch creds directory if it exists
	if info, err := os.Stat(natsCredsDir); err == nil && info.IsDir() {
		go watch.WatchDir(ctx, natsCredsDir, debounce, func() {
			if err := server.Reload(); err != nil {
				log.Printf("Reload after creds dir change: %v", err)
			}
		})
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
