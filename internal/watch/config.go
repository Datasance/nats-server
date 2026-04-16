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

package watch

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 500 * time.Millisecond

// WatchConfigFile watches the config file at configPath for changes. On write/create
// (after debounce), it calls onReload. Runs until ctx is cancelled.
// The parent directory of configPath must exist (e.g. volume-mounted).
func WatchConfigFile(ctx context.Context, configPath string, debounce time.Duration, onReload func()) {
	if debounce <= 0 {
		debounce = defaultDebounce
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("ERROR: Failed to create fsnotify watcher for config file: %v", err)
		return
	}
	defer watcher.Close()

	dir := filepath.Dir(configPath)
	if err := watcher.Add(dir); err != nil {
		log.Printf("ERROR: Failed to add watch on %s: %v", dir, err)
		return
	}

	var debounceTimer *time.Timer
	var debounceMu sync.Mutex
	scheduleReload := func() {
		debounceMu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(debounce, func() {
			onReload()
		})
		debounceMu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Clean(event.Name) != filepath.Clean(configPath) {
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				scheduleReload()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("ERROR: Config file watcher error: %v", err)
		}
	}
}

// FileExists returns true if path exists and is a regular file (or symlink to file).
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
