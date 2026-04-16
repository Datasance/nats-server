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

package jwtcopy

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SyncMountToJWT makes NATS_JWT_DIR mirror NATS_JWT_MOUNT_DIR: copies all *.jwt files
// from mountDir to jwtDir (overwrite), then removes any *.jwt in jwtDir not in mountDir.
// No directory renames—only file copy and remove—so it is safe when jwtDir is a volume
// mount (no cross-device link). If mountDir and jwtDir are the same path, skips and
// returns 0, 0, nil. If mountDir does not exist, returns 0, 0, nil. Empty mount list
// returns without removing anything (e.g. K8s ConfigMap rotation).
func SyncMountToJWT(mountDir, jwtDir string) (copied int, removed int, err error) {
	if filepath.Clean(mountDir) == filepath.Clean(jwtDir) {
		return 0, 0, nil
	}
	mountNames, err := listJWTFileNames(mountDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if len(mountNames) == 0 {
		return 0, 0, nil
	}
	if err := os.MkdirAll(jwtDir, 0755); err != nil {
		return 0, 0, err
	}
	for _, name := range mountNames {
		src := filepath.Join(mountDir, name)
		dst := filepath.Join(jwtDir, name)
		if err := copyFile(src, dst); err != nil {
			return copied, removed, err
		}
		copied++
	}
	mountSet := make(map[string]struct{}, len(mountNames))
	for _, n := range mountNames {
		mountSet[n] = struct{}{}
	}
	jwtNames, err := listJWTFileNames(jwtDir)
	if err != nil {
		return copied, removed, err
	}
	for _, name := range jwtNames {
		if _, inMount := mountSet[name]; inMount {
			continue
		}
		if err := os.Remove(filepath.Join(jwtDir, name)); err != nil && !os.IsNotExist(err) {
			return copied, removed, err
		}
		removed++
	}
	return copied, removed, nil
}

// listJWTFileNames returns base names of *.jwt files in dir, skipping *.jwt.delete.
// Same convention as jspurge.AccountsFromJWTDir.
func listJWTFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".jwt") {
			continue
		}
		if strings.HasSuffix(name, ".delete") {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
