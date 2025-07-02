package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/berquerant/k8s-object-diff-go/version"
	"github.com/berquerant/structconfig"
	"github.com/spf13/pflag"
)

const usage = `objdiff - k8s object diff by object id

# Usage

  objdiff [flags] LEFT_FILE RIGHT_FILE

# Object ID

A unique ID for a k8s object.
e.g.

  apiVersion: v1
  kind: Pod
  metadata:
    name: nginx
    namespace: default

then id is 'v1>Pod>default>nginx'.

# Output format
## id

All object IDs.

## text

Unified diff.

## yaml

Array of

  id: "Object ID"
  diff: "Unified diff"
  left: "Left object (optional)"
  right: "Right object (optional)"

# Exit status

0 if inputs are the same.
1 if inputs differ.
Otherwise 2.

# Flags`

type Config struct {
	Context           int    `name:"context" short:"C" default:"3" usage:"diff context"`
	Separator         string `name:"separator" short:"d" default:">" usage:"object id separator"`
	Indent            int    `name:"indent" short:"n" default:"2" usage:"yaml indent"`
	Out               string `name:"out" short:"o" default:"text" usage:"output format: text,yaml,id"`
	Debug             bool   `name:"debug" usage:"enable debug log"`
	Color             bool   `name:"color" short:"c" usage:"colored diff"`
	DiffSuccess       bool   `name:"success" usage:"exit with 0 even if inputs differ"`
	AllowDuplicateKey bool   `name:"allowDuplicateKey" default:"true" usage:"allow the use of keys with the same name in the same map"`
}

type outMode string

const (
	outModeUnknown outMode = "unknown"
	outModeText    outMode = "text"
	outModeYaml    outMode = "yaml"
	outModeID      outMode = "id"
)

func (c *Config) outMode() outMode {
	switch c.Out {
	case string(outModeText):
		return outModeText
	case string(outModeYaml):
		return outModeYaml
	case string(outModeID):
		return outModeID
	default:
		return outModeUnknown
	}
}

const (
	exitCodeDiffFound = 1
	exitCodeFailure   = 2
)

func main() {
	fs := pflag.NewFlagSet("main", pflag.ContinueOnError)
	fs.Usage = func() {
		fmt.Println(usage)
		fs.PrintDefaults()
	}

	fs.Bool("version", false, "print objdiff version")
	c, err := structconfig.NewConfigWithMerge(
		structconfig.New[Config](),
		structconfig.NewMerger[Config](),
		fs,
	)
	if errors.Is(err, pflag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	setupLogger(os.Stderr, c.Debug)

	if v, _ := fs.GetBool("version"); v {
		version.Write(os.Stdout)
		return
	}

	if fs.NArg() != 3 {
		slog.Error("2 files are required")
		os.Exit(exitCodeFailure)
	}
	if c.outMode() == outModeUnknown {
		slog.Error("invalid out", slog.String("out", c.Out))
		os.Exit(exitCodeFailure)
	}

	if err := run(c, fs.Arg(1), fs.Arg(2)); err != nil {
		if errors.Is(err, errDiffFound) {
			if c.DiffSuccess {
				return
			}
			os.Exit(exitCodeDiffFound)
		}
		os.Exit(exitCodeFailure)
	}
}

func setupLogger(w io.Writer, debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
