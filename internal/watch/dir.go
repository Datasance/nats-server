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
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchDir watches basePath (and immediate subdirs) for changes. On any create/write/remove
// (after debounce), it calls onReload. Runs until ctx is cancelled.
// If basePath does not exist, the watcher returns without error (caller may start it when dir appears).
func WatchDir(ctx context.Context, basePath string, debounce time.Duration, onReload func()) {
	if debounce <= 0 {
		debounce = defaultDebounce
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("ERROR: Failed to create fsnotify watcher for dir %s: %v", basePath, err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(basePath); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("ERROR: Failed to add watch on %s: %v", basePath, err)
		}
		return
	}

	subdirs := make(map[string]struct{})
	var mu sync.Mutex
	addSubdir := func(path string) {
		if path == basePath {
			return
		}
		rel, err := filepath.Rel(basePath, path)
		if err != nil || rel == "" || rel == ".." || strings.HasPrefix(rel, "..") {
			return
		}
		if filepath.Dir(rel) != "." {
			return
		}
		mu.Lock()
		if _, ok := subdirs[path]; !ok {
			subdirs[path] = struct{}{}
			_ = watcher.Add(path)
		}
		mu.Unlock()
	}

	// Initial scan of subdirs
	if entries, err := os.ReadDir(basePath); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				addSubdir(filepath.Join(basePath, e.Name()))
			}
		}
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
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove) != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() && event.Op == fsnotify.Create {
					addSubdir(event.Name)
				}
				scheduleReload()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("ERROR: Dir watcher error: %v", err)
		}
	}
}
