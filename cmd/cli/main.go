package main

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func main() {
	flag.CommandLine.SetOutput(io.Discard)
	containerID := flag.String("container-id", "", "")
	flag.Parse()

	if *containerID == "" {
		os.Exit(2)
	}

	cmd := buildMCPCommand(*containerID)
	if err := runWithStdio(cmd); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func buildMCPCommand(containerID string) *exec.Cmd {
	execID := "mcp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if runtime.GOOS == "darwin" {
		return exec.Command(
			"limactl",
			"shell",
			"--tty=false",
			"default",
			"--",
			"sudo",
			"-n",
			"ctr",
			"-n",
			"default",
			"tasks",
			"exec",
			"--exec-id",
			execID,
			containerID,
			"/mcp",
		)
	}
	return exec.Command(
		"ctr",
		"-n",
		"default",
		"tasks",
		"exec",
		"--exec-id",
		execID,
		containerID,
		"/mcp",
	)
}

func runWithStdio(cmd *exec.Cmd) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return err
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		return err
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(stdin, os.Stdin)
		_ = stdin.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stdout, stdout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stderr, stderr)
	}()

	err = cmd.Wait()
	wg.Wait()
	return err
}
