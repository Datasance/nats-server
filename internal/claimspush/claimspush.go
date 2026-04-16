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

package claimspush

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/datasance/nats-server/internal/jspurge"
	"github.com/nats-io/nats.go"
)

const (
	claimsUpdateSubject = "$SYS.REQ.CLAIMS.UPDATE"
	defaultTimeout      = 10 * time.Second
)

// PushAccountJWTs lists account JWT files in jwtDir (same convention as jspurge: account-pub-key.jwt),
// connects to the NATS server at clientURL with credsPath (system account, same as jspurge),
// and sends a request to $SYS.REQ.CLAIMS.UPDATE with each account's raw JWT. Single server only.
// If credsPath is empty, returns immediately without error (same as runJetStreamReconcile).
// Logs per-account success/failure and a summary; non-fatal errors do not stop the process.
func PushAccountJWTs(ctx context.Context, jwtDir, clientURL, credsPath string, timeout time.Duration) {
	if credsPath == "" {
		return
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	accounts, err := jspurge.AccountsFromJWTDir(jwtDir)
	if err != nil {
		log.Printf("Claims update: failed to list JWT dir %s: %v", jwtDir, err)
		return
	}
	if len(accounts) == 0 {
		log.Printf("Claims update: no account JWTs in %s", jwtDir)
		return
	}

	opts := []nats.Option{nats.UserCredentials(credsPath)}
	nc, err := nats.Connect(clientURL, opts...)
	if err != nil {
		log.Printf("Claims update: failed to connect to %s: %v", clientURL, err)
		return
	}
	defer nc.Close()

	var pushed, failed int
	for _, account := range accounts {
		jwtPath := filepath.Join(jwtDir, account+".jwt")
		raw, err := os.ReadFile(jwtPath)
		if err != nil {
			log.Printf("Claims update: failed to read %s: %v", jwtPath, err)
			failed++
			continue
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		_, err = nc.RequestWithContext(reqCtx, claimsUpdateSubject, raw)
		cancel()
		if err != nil {
			log.Printf("Claims update: failed for account %s: %v", account, err)
			failed++
			continue
		}
		pushed++
	}
	log.Printf("Claims update: pushed %d account JWTs to %s (failed: %d)", pushed, clientURL, failed)
}
