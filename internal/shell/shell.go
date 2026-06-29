package shell

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result holds the output of a shell command execution.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
}

// RunCommand executes a shell command and captures its output.
func RunCommand(command string, timeoutSeconds int) (*Result, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	start := time.Now()

	cmd := exec.Command("bash", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start).Round(time.Millisecond).String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("execution error: %w", err)
		}
	}

	result := &Result{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
		Duration: duration,
	}

	return result, nil
}

// RunCommandInDir runs a shell command in a specific directory.
func RunCommandInDir(command, dir string, timeoutSeconds int) (*Result, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	start := time.Now()

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start).Round(time.Millisecond).String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("execution error: %w", err)
		}
	}

	result := &Result{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
		Duration: duration,
	}

	return result, nil
}
