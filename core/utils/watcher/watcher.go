package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/utils/logger"
)

func changesDetected(dir string, since time.Time) bool {
	var changed bool

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().After(since) {
			changed = true
		}
		return nil
	})

	return changed
}

type BuildFunc func() error

func ExecCommand(command string, args ...string) BuildFunc {
	return func() error {
		cmd := exec.Command(command, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error building: \"%s %s\": %w", command, strings.Join(args, " "), err)
		}
		return nil
	}
}

type RunFunc func() (stop func(), err error)

func LaunchCommand(command string, args ...string) RunFunc {
	return func() (func(), error) {
		cmd := exec.Command(command, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("error running: \"%s %s\": %w", command, strings.Join(args, " "), err)
		}
		return func() {
			cmd.Process.Kill()
		}, nil
	}
}

func Run(before []BuildFunc, run RunFunc, after []BuildFunc) (func(), error) {
	for _, fn := range before {
		err := fn()
		if err != nil {
			return nil, err
		}
	}
	stop, err := run()
	if err != nil {
		return nil, err
	}
	for _, fn := range after {
		err := fn()
		if err != nil {
			stop()
			return nil, err
		}
	}
	return stop, nil
}

func Watch(every time.Duration, dirs ...string) {
	if every == 0 {
		every = 500 * time.Millisecond
	}

	if len(dirs) == 0 {
		dirs = append(dirs, ".")
	}

	var stop func()
	var err error
	var lastScan time.Time

	for {
		canSleep := false
		for _, dir := range dirs {
			if !changesDetected(dir, lastScan) {
				canSleep = true
			}
		}
		if canSleep {
			time.Sleep(every)
			continue
		}

		if stop != nil {
			logger.Printfs("ylChange detected")
			logger.Printfs("ylRestarting...")
			stop()
		}

		stop, err = Run(
			[]BuildFunc{ExecCommand("go", "build", "-o", "temp")},
			LaunchCommand("./temp"),
			nil,
		)
		logger.Printfs("grReady")
		logger.CheckError(err)
		lastScan = time.Now()
		time.Sleep(every)
	}
}