package exec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CommandExists checks if a command is available in PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Run executes a command with stdout/stderr connected to the terminal.
func Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunInDir executes a command in a specific directory.
func RunInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunPawlWorkflow copies the named workflow to .pawl/config.json and runs it.
func RunPawlWorkflow(siteDir, workflow string) error {
	src := filepath.Join(siteDir, ".pawl", "workflows", workflow+".json")
	dst := filepath.Join(siteDir, ".pawl", "config.json")
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read workflow %q: %w", workflow, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write pawl config: %w", err)
	}
	return RunInDir(siteDir, "pawl", "start", workflow)
}

// Output executes a command and returns its stdout as a string.
func Output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}
