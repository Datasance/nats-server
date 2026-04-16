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

package exec

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// Start starts the command with the given args and optional extra environment variables.
// The process environment is always preserved: cmd.Env = append(os.Environ(), extraEnv...),
// so that placeholders like $SERVER_NAME in config files are resolved by the child.
// workDir is the working directory for the process (e.g. config file's directory); empty means current dir.
// Stdout and stderr are forwarded to the parent's output. The returned *exec.Cmd can be used
// to send signals (e.g. SIGHUP for config reload) via cmd.Process.Signal(syscall.SIGHUP).
// The caller must run cmd.Wait() in a goroutine and send the result to an exit channel.
func Start(name string, args []string, extraEnv []string, workDir string) (*exec.Cmd, error) {
	log.Printf("Starting command: %s with args: %v", name, args)

	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		scanner := bufio.NewScanner(outReader)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	errReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		scanner := bufio.NewScanner(errReader)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
