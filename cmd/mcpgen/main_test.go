package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/mcpgen"
	"github.com/stretchr/testify/require"
)

func TestRunSuccess(t *testing.T) {
	var gotCfg mcpgen.GeneratorConfig
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	originalGenerate := generate
	generate = func(cfg mcpgen.GeneratorConfig) error {
		gotCfg = cfg
		return nil
	}
	t.Cleanup(func() {
		generate = originalGenerate
	})

	err := run([]string{"-fields", "fields", "-v2", "v2", "-out", "out"}, logger)
	require.NoError(t, err)
	require.Equal(t, "fields", gotCfg.FieldsDir)
	require.Equal(t, "v2", gotCfg.V2Dir)
	require.Equal(t, "out", gotCfg.OutDir)
	require.Contains(t, buf.String(), "Generated MCP tools to out")
}

func TestRunReturnsGenerateError(t *testing.T) {
	expectedErr := errors.New("boom")

	originalGenerate := generate
	generate = func(cfg mcpgen.GeneratorConfig) error {
		return expectedErr
	}
	t.Cleanup(func() {
		generate = originalGenerate
	})

	err := run([]string{}, log.New(&bytes.Buffer{}, "", 0))
	require.ErrorIs(t, err, expectedErr)
}

func TestRunReturnsFlagError(t *testing.T) {
	err := run([]string{"-unknown"}, log.New(&bytes.Buffer{}, "", 0))
	require.Error(t, err)
}

func TestMainCallsFatalOnError(t *testing.T) {
	expectedErr := errors.New("boom")
	called := false

	originalGenerate := generate
	originalFatal := fatal
	originalArgs := os.Args
	generate = func(cfg mcpgen.GeneratorConfig) error {
		return expectedErr
	}
	fatal = func(args ...any) {
		called = true
		require.Len(t, args, 1)
		require.ErrorIs(t, args[0].(error), expectedErr)
	}
	os.Args = []string{"mcpgen"}
	t.Cleanup(func() {
		generate = originalGenerate
		fatal = originalFatal
		os.Args = originalArgs
	})

	main()
	require.True(t, called)
}

func TestMainSuccess(t *testing.T) {
	called := false
	buf := &bytes.Buffer{}
	originalOutput := log.Default().Writer()
	log.SetOutput(buf)

	originalGenerate := generate
	originalFatal := fatal
	originalArgs := os.Args
	generate = func(cfg mcpgen.GeneratorConfig) error {
		return nil
	}
	fatal = func(args ...any) {
		called = true
	}
	os.Args = []string{"mcpgen", "-out", "out"}
	t.Cleanup(func() {
		generate = originalGenerate
		fatal = originalFatal
		os.Args = originalArgs
		log.SetOutput(originalOutput)
	})

	main()
	require.False(t, called)
	require.Contains(t, buf.String(), "Generated MCP tools to out")
}
