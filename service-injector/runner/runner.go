package runner

import (
	"bytes"
	"fmt"
	"os/exec"
)

// BuildResult holds the outcome of go build.
type BuildResult struct {
	Success bool
	Output  string
	Error   string
}

// GoFmt runs gofmt -w . in the project directory.
func GoFmt(dir string) error {
	cmd := exec.Command("gofmt", "-w", ".")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofmt: %v\n%s", err, stderr.String())
	}
	return nil
}

// GoBuild runs go build ./... in the project directory.
func GoBuild(dir string) BuildResult {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return BuildResult{false, stdout.String(),
			fmt.Sprintf("go build: %v\n%s", err, stderr.String())}
	}
	return BuildResult{true, stdout.String(), ""}
}

// GoTidy runs go mod tidy in the project directory.
func GoTidy(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %v\n%s", err, stderr.String())
	}
	return nil
}
