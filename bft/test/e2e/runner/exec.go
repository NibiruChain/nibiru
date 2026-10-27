package main

import (
	"bytes"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
)

// execute executes a shell command.
func exec(args ...string) error {
	_, err := execOutput(args...)
	return err
}

func execOutput(args ...string) ([]byte, error) {
	cmd := osexec.Command(args[0], args[1:]...) //nolint:gosec
	out, err := cmd.CombinedOutput()
	switch err := err.(type) {
	case nil:
		return out, nil
	case *osexec.ExitError:
		return nil, fmt.Errorf("failed to run %q:\n%v", args, string(out))
	default:
		return nil, err
	}
}

// execVerbose executes a shell command while displaying its output.
func execVerbose(args ...string) error {
	cmd := osexec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// execCompose runs a Docker Compose command for a testnet.
func execCompose(dir string, args ...string) error {
	return exec(append(
		[]string{"docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml")},
		args...)...)
}

// execComposeOutput runs a Docker Compose command for a testnet and returns
// its stdout only. Compose prints warnings to stderr (for example that the
// `version` attribute is obsolete), and callers check for empty output.
func execComposeOutput(dir string, args ...string) ([]byte, error) {
	args = append(
		[]string{"compose", "-f", filepath.Join(dir, "docker-compose.yml")},
		args...)
	var stderr bytes.Buffer
	cmd := osexec.Command("docker", args...) //nolint:gosec
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	switch err := err.(type) {
	case nil:
		return out, nil
	case *osexec.ExitError:
		return nil, fmt.Errorf("failed to run %q:\n%s%s",
			append([]string{"docker"}, args...), out, stderr.Bytes())
	default:
		return nil, err
	}
}

// execComposeVerbose runs a Docker Compose command for a testnet and displays its output.
func execComposeVerbose(dir string, args ...string) error {
	return execVerbose(append(
		[]string{"docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml")},
		args...)...)
}

// execDocker runs a Docker command.
func execDocker(args ...string) error {
	return exec(append([]string{"docker"}, args...)...)
}
