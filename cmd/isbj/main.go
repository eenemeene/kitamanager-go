package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eenemeene/kitamanager-go/internal/isbj"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: isbj <path-to-xlsx>\n")
		os.Exit(1)
	}

	f, err := os.Open(filepath.Clean(os.Args[1])) //nolint:gosec // G703: CLI tool intentionally opens user-specified file
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	raw, err := isbj.ParseFromReader(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	converted, err := isbj.Convert(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "convert: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(converted); err != nil {
		fmt.Fprintf(os.Stderr, "json: %v\n", err)
		os.Exit(1)
	}
}
