//go:build windows

package workflow

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const taskkillTimeout = 5 * time.Second

type taskkillProcess interface {
	Start() error
	Wait() error
}

var newTaskkillProcess = func(ctx context.Context, pid int) taskkillProcess {
	killer := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	killer.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return killer
}

func configureProcessTree(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		ctx, cancel := context.WithTimeout(context.Background(), taskkillTimeout)
		killer := newTaskkillProcess(ctx, command.Process.Pid)
		if err := killer.Start(); err != nil {
			cancel()
			return killCommandProcess(command)
		}
		go func() {
			defer cancel()
			if err := killer.Wait(); err != nil {
				_ = killCommandProcess(command)
			}
		}()
		return nil
	}
}

func killCommandProcess(command *exec.Cmd) error {
	if err := command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}
