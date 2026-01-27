package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var execCommand = exec.Command

func main() {
	outPath := flag.String("out", ".tmp/unifi-controller-version", "Output path")
	flag.Parse()

	if err := run(*outPath); err != nil {
		fail(err)
	}
}

func run(outPath string) error {
	version, err := goUnifiModuleVersion()
	if err != nil {
		return err
	}

	if err := ensureModuleDownloaded(version); err != nil {
		return err
	}

	modCache, err := goModCache()
	if err != nil {
		return err
	}

	controllerVersion, err := readControllerVersion(modCache, version)
	if err != nil {
		return err
	}

	if err := writeOutput(outPath, controllerVersion); err != nil {
		return err
	}

	return nil
}

func goUnifiModuleVersion() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	re := regexp.MustCompile(`(?m)^\s*github.com/filipowm/go-unifi\s+([^\s]+)`)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return "", errors.New("go-unifi module version not found in go.mod")
	}

	return strings.TrimSpace(string(match[1])), nil
}

func ensureModuleDownloaded(version string) error {
	cmd := execCommand("go", "mod", "download", "github.com/filipowm/go-unifi@"+version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func goModCache() (string, error) {
	if cache := strings.TrimSpace(os.Getenv("GOMODCACHE")); cache != "" {
		return cache, nil
	}

	cmd := execCommand("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMODCACHE: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func readControllerVersion(modCache, moduleVersion string) (string, error) {
	versionPath := filepath.Join(
		modCache,
		"github.com",
		"filipowm",
		"go-unifi@"+moduleVersion,
		"unifi",
		"version.generated.go",
	)

	data, err := os.ReadFile(versionPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", versionPath, err)
	}

	re := regexp.MustCompile(`UnifiVersion\s*=\s*"([^"]+)"`)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return "", errors.New("UnifiVersion not found in version.generated.go")
	}

	return strings.TrimSpace(string(match[1])), nil
}

func writeOutput(path string, version string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return errors.New("empty controller version")
	}

	return os.WriteFile(path, append(bytes.TrimSpace([]byte(trimmed)), '\n'), 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
