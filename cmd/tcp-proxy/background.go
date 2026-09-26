package main

import (
	"os"
	"os/exec"
)

const backgroundChildEnv = "GOGETWAY_BACKGROUND_CHILD"

func isBackgroundChild() bool {
	return os.Getenv(backgroundChildEnv) == "1"
}

func startBackgroundProcess(args []string) (int, error) {
	executable, err := os.Executable()
	if err != nil {
		return 0, err
	}

	nullDevice, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer nullDevice.Close()

	cmd := exec.Command(executable, args...)
	cmd.Env = append(os.Environ(), backgroundChildEnv+"=1")
	cmd.Stdin = nullDevice
	cmd.Stdout = nullDevice
	cmd.Stderr = nullDevice
	configureBackgroundProcess(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return 0, err
	}
	return pid, nil
}
