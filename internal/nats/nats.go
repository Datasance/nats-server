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

package nats

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/datasance/nats-server/internal/config"
	execpkg "github.com/datasance/nats-server/internal/exec"
)

type Server struct {
	cmd *exec.Cmd
	mu  sync.Mutex
}

// Start starts nats-server with the given server config file path. The process environment
// is preserved so that placeholders like $SERVER_NAME in the config are resolved. workDir
// is set to the config file's directory so relative paths (e.g. include) resolve. When the
// process exits, the error (if any) is sent to exitCh.
func (s *Server) Start(serverConfPath string, exitCh chan<- error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil {
		return fmt.Errorf("nats-server already started")
	}

	bin := config.GetNatsServerBin()
	workDir := filepath.Dir(serverConfPath)
	args := []string{"-c", serverConfPath}
	if port := config.GetNatsMonitorPort(); port > 0 {
		args = append(args, "-m", strconv.Itoa(port))
	}

	cmd, err := execpkg.Start(bin, args, nil, workDir)
	if err != nil {
		return fmt.Errorf("failed to start nats-server: %w", err)
	}
	s.cmd = cmd

	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		s.cmd = nil
		s.mu.Unlock()
		if exitCh != nil {
			exitCh <- err
		}
	}()

	log.Printf("NATS server started with config %s", serverConfPath)
	return nil
}

// Reload sends SIGHUP to the running nats-server process so it reloads config and certs.
func (s *Server) Reload() error {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("nats-server not running")
	}
	if err := cmd.Process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP: %w", err)
	}
	log.Printf("Sent SIGHUP to nats-server for config reload")
	return nil
}

// Stop sends SIGINT to the running nats-server process for graceful shutdown.
// The process will exit; the caller should wait for the exit on exitCh and may then call Start again (restart).
func (s *Server) Stop() error {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("nats-server not running")
	}
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("failed to send SIGINT: %w", err)
	}
	log.Printf("Sent SIGINT to nats-server for graceful stop (restart)")
	return nil
}
