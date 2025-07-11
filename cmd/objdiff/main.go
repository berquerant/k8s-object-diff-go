package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/internal"
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
## idlist

All object IDs.

## id

ID diff.

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

# Override differ

  objdiff -x diff left.yml right.yml
invokes
  diff --unified=3 --color=never --label left.yml --label right.yml LEFT_FILE RIGHT_FILE

  DIFFCMD='diff' objdiff -c -C 5 left.yml right.yml
invokes
  diff --unified=5 --color=always --label left.yml --label right.yml LEFT_FILE RIGHT_FILE

# Flags`

type Config struct {
	Context           int    `name:"context" short:"C" default:"3" usage:"diff context"`
	Separator         string `name:"separator" short:"d" default:">" usage:"object id separator"`
	Indent            int    `name:"indent" short:"n" default:"2" usage:"yaml indent"`
	Out               string `name:"out" short:"o" default:"text" usage:"output format: text,yaml,id,idlist"`
	Debug             bool   `name:"debug" usage:"enable debug log"`
	Quiet             bool   `name:"quiet" short:"q" usage:"quiet log"`
	Color             bool   `name:"color" short:"c" usage:"colored diff"`
	DiffSuccess       bool   `name:"success" usage:"exit with 0 even if inputs differ"`
	AllowDuplicateKey bool   `name:"allowDuplicateKey" default:"true" usage:"allow the use of keys with the same name in the same map"`
	DiffCommand       string `name:"diffCmd" short:"x" usage:"invoke this to get diff instead of builtin differ"`
}

type outMode string

const (
	outModeUnknown outMode = "unknown"
	outModeText    outMode = "text"
	outModeYaml    outMode = "yaml"
	outModeID      outMode = "id"
	outModeIDList  outMode = "idlist"
)

func (c *Config) outMode() outMode {
	switch c.Out {
	case string(outModeText):
		return outModeText
	case string(outModeYaml):
		return outModeYaml
	case string(outModeID):
		return outModeID
	case string(outModeIDList):
		return outModeIDList
	default:
		return outModeUnknown
	}
}

var errNoDiffCommand = errors.New("NoDiffCommand")

func (c *Config) diffCommand() ([]string, error) {
	if len(c.DiffCommand) == 0 {
		return nil, errNoDiffCommand
	}
	xs := strings.Split(internal.EscapeCommand(c.DiffCommand), " ")
	if len(xs) == 0 {
		panic("unreachable: diffCommand specified but no command")
	}
	head, err := exec.LookPath(xs[0])
	if err != nil {
		return nil, fmt.Errorf("lookup %s: %w", xs[0], err)
	}
	xs[0] = head
	return xs, nil
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

	setupLogger(os.Stderr, c.Debug, c.Quiet)

	if v, _ := fs.GetBool("version"); v {
		version.Write(os.Stdout)
		return
	}
	if c.Context < 0 {
		slog.Error("invalid context length")
		os.Exit(exitCodeFailure)
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
		slog.Error("exit", slog.Any("err", err))
		os.Exit(exitCodeFailure)
	}
}

func setupLogger(w io.Writer, debug, quiet bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	if quiet {
		level = slog.LevelError
	}
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
