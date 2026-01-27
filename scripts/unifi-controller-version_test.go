package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainHappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoMod(t, tmpDir, "v1.2.3")
	modCache := filepath.Join(tmpDir, "modcache")
	writeVersionFile(t, modCache, "v1.2.3", "9.9.9")

	withTempDir(t, tmpDir, func() {
		execCommand = func(string, ...string) *exec.Cmd {
			return exec.Command("true")
		}
		defer func() { execCommand = exec.Command }()

		t.Setenv("GOMODCACHE", modCache)

		outPath := filepath.Join(tmpDir, "out", "version")
		resetFlags(t, []string{"cmd", "-out", outPath})

		main()

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if string(content) != "9.9.9\n" {
			t.Fatalf("expected version output, got %q", string(content))
		}
	})
}

func TestRunMissingGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	withTempDir(t, tmpDir, func() {
		if err := run("output"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRunDownloadError(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoMod(t, tmpDir, "v1.2.3")

	withTempDir(t, tmpDir, func() {
		execCommand = func(string, ...string) *exec.Cmd {
			return exec.Command("false")
		}
		defer func() { execCommand = exec.Command }()

		if err := run("output"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRunGoModCacheError(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoMod(t, tmpDir, "v1.2.3")
	writeVersionFile(t, filepath.Join(tmpDir, "modcache"), "v2.0.0", "9.9.9")

	withTempDir(t, tmpDir, func() {
		execCommand = func(name string, args ...string) *exec.Cmd {
			if name == "go" && len(args) > 1 && args[0] == "mod" && args[1] == "download" {
				return exec.Command("true")
			}
			return exec.Command("false")
		}
		defer func() { execCommand = exec.Command }()

		t.Setenv("GOMODCACHE", "")
		if err := run("output"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRunWriteOutputError(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoMod(t, tmpDir, "v1.2.3")
	modCache := filepath.Join(tmpDir, "modcache")
	writeVersionFile(t, modCache, "v1.2.3", "9.9.9")
	blocker := filepath.Join(tmpDir, "block")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	withTempDir(t, tmpDir, func() {
		execCommand = func(string, ...string) *exec.Cmd {
			return exec.Command("true")
		}
		defer func() { execCommand = exec.Command }()

		t.Setenv("GOMODCACHE", modCache)
		if err := run(filepath.Join(blocker, "file")); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGoUnifiModuleVersion(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoMod(t, tmpDir, "v1.8.1")

	withTempDir(t, tmpDir, func() {
		version, err := goUnifiModuleVersion()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != "v1.8.1" {
			t.Fatalf("expected v1.8.1, got %q", version)
		}
	})
}

func TestGoUnifiModuleVersionMissing(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	withTempDir(t, tmpDir, func() {
		if _, err := goUnifiModuleVersion(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGoModCacheFromEnv(t *testing.T) {
	t.Setenv("GOMODCACHE", "/tmp/go-cache")

	cache, err := goModCache()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache != "/tmp/go-cache" {
		t.Fatalf("expected /tmp/go-cache, got %q", cache)
	}
}

func TestGoModCacheFromCommand(t *testing.T) {
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf /tmp/cache")
	}
	defer func() { execCommand = exec.Command }()

	t.Setenv("GOMODCACHE", "")
	cache, err := goModCache()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache != "/tmp/cache" {
		t.Fatalf("expected /tmp/cache, got %q", cache)
	}
}

func TestReadControllerVersion(t *testing.T) {
	tmpDir := t.TempDir()
	writeVersionFile(t, tmpDir, "v1.2.3", "9.0.114")

	version, err := readControllerVersion(tmpDir, "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "9.0.114" {
		t.Fatalf("expected 9.0.114, got %q", version)
	}
}

func TestReadControllerVersionMissing(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "github.com", "filipowm", "go-unifi@v1.2.3", "unifi")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "version.generated.go"), []byte("package unifi\n"), 0o644); err != nil {
		t.Fatalf("write version file: %v", err)
	}

	if _, err := readControllerVersion(tmpDir, "v1.2.3"); err == nil {
		t.Fatal("expected error")
	}
}

func TestReadControllerVersionMissingFile(t *testing.T) {
	if _, err := readControllerVersion(t.TempDir(), "v1.2.3"); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteOutput(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "out", "version")

	if err := writeOutput(path, "9.0.114"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "9.0.114\n" {
		t.Fatalf("expected newline, got %q", string(data))
	}
}

func TestWriteOutputEmpty(t *testing.T) {
	if err := writeOutput("/tmp/unused", " "); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteOutputMkdirFailure(t *testing.T) {
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "block")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	path := filepath.Join(blocker, "file")
	if err := writeOutput(path, "9.0.114"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMainError(t *testing.T) {
	if os.Getenv("TEST_MAIN_ERROR") == "1" {
		resetFlags(t, []string{"cmd"})
		main()
		return
	}

	binary, execErr := os.Executable()
	if execErr != nil {
		t.Fatalf("executable: %v", execErr)
	}

	cmd := exec.Command(binary, "-test.run=TestMainError")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "TEST_MAIN_ERROR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit, got %v", err)
	}
	if !strings.Contains(stderr.String(), "read go.mod") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func resetFlags(t *testing.T, args []string) {
	t.Helper()
	oldArgs := os.Args
	oldFlags := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ExitOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = args
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlags
	})
}

func writeGoMod(t *testing.T, dir, version string) {
	t.Helper()
	content := []byte("module example\n\nrequire (\n\tgithub.com/filipowm/go-unifi " + version + "\n)\n")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), content, 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func writeVersionFile(t *testing.T, modCache, version, controller string) {
	t.Helper()
	path := filepath.Join(modCache, "github.com", "filipowm", "go-unifi@"+version, "unifi")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := bytes.NewBuffer(nil)
	content.WriteString("package unifi\n\nconst UnifiVersion = \"")
	content.WriteString(controller)
	content.WriteString("\"\n")
	if err := os.WriteFile(filepath.Join(path, "version.generated.go"), content.Bytes(), 0o644); err != nil {
		t.Fatalf("write version file: %v", err)
	}
}

func withTempDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			panic(err)
		}
	}()
	fn()
}
