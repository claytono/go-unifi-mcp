// mcpgen generates MCP tool handlers from UniFi API field definitions.
package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/claytono/go-unifi-mcp/internal/mcpgen"
)

var generate = mcpgen.Generate
var fatal = log.Fatal

func main() {
	if err := run(os.Args[1:], log.Default()); err != nil {
		fatal(err)
	}
}

func run(args []string, logger *log.Logger) error {
	flagSet := flag.NewFlagSet("mcpgen", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	fieldsDir := flagSet.String("fields", ".tmp/fields", "Path to v1 field definitions")
	v2Dir := flagSet.String("v2", "internal/gounifi/v2", "Path to v2 field definitions")
	outDir := flagSet.String("out", "internal/tools/generated", "Output directory")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	cfg := mcpgen.GeneratorConfig{
		FieldsDir: *fieldsDir,
		V2Dir:     *v2Dir,
		OutDir:    *outDir,
	}

	if err := generate(cfg); err != nil {
		return err
	}

	logger.Printf("Generated MCP tools to %s", *outDir)
	return nil
}
