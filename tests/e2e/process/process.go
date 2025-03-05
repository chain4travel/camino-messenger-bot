// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	processTickerInterval = 50 * time.Millisecond
	killTimeout           = 2 * time.Second
)

func ListenForProcessError(cmd *exec.Cmd) chan error {
	errChan := make(chan error)
	go func() {
		if err := cmd.Wait(); err != nil {
			if err.Error() != "signal: killed" {
				errChan <- fmt.Errorf("process %d exited with error: %w", cmd.Process.Pid, err)
			}
		}
		close(errChan)
	}()
	return errChan
}

func StopProcess(ctx context.Context, pid int) error {
	process, err := getProcess(pid)
	switch {
	case err != nil:
		return err
	case process == nil:
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to process with pid %d: %w", pid, err)
	}

	if err := waitKillProcess(ctx, pid); err != nil {
		return fmt.Errorf("failed to wait for process with pid %d to stop: %w", pid, err)
	}

	return nil
}

func waitKillProcess(ctx context.Context, pid int) error {
	ticker := time.NewTicker(processTickerInterval)
	defer ticker.Stop()

	killTimeoutCtx, cancelKillTimeout := context.WithTimeout(ctx, killTimeout)
	defer cancelKillTimeout()

	for {
		process, err := getProcess(pid)
		switch {
		case err != nil:
			return fmt.Errorf("failed to retrieve process %d: %w", pid, err)
		case process == nil:
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to see process %d stop: %w", pid, ctx.Err())
		case <-killTimeoutCtx.Done():
			// Process did not stop in time (killTimeout), killing it
			if err := process.Signal(syscall.SIGKILL); err != nil {
				if errors.Is(err, os.ErrProcessDone) {
					return nil // race condition fix: It's already done
				}
				return fmt.Errorf("failed to send SIGKILL to process with pid %d: %w", pid, err)
			}
			// The timeout is done - prevent it from triggering again
			cancelKillTimeout()
		case <-ticker.C:
		}
	}
}

// Might misbehave on Windows systems because of different behavior of os.FindProcess
func getProcess(pid int) (*os.Process, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("could not find process %d: %w", pid, err)
	}

	// Sending 0 will not actually send a signal but will perform error checking.
	err = process.Signal(syscall.Signal(0))
	switch {
	case err == nil:
		// Process is running
		return process, nil
	case errors.Is(err, os.ErrProcessDone):
		// Process is not running
		return nil, nil
	}

	return nil, fmt.Errorf("failed to check if process %d is running: %w", pid, err)
}
