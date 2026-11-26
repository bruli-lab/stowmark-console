package desktop

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Open(path string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}

	process := exec.Command(command, path)
	if err := process.Start(); err != nil {
		return fmt.Errorf("open %s with %s: %w", path, command, err)
	}

	go func() { _ = process.Wait() }()

	return nil
}
