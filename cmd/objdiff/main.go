package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/berquerant/k8s-object-diff-go/config"
	"github.com/berquerant/k8s-object-diff-go/version"
	"github.com/berquerant/structconfig"
	"github.com/spf13/pflag"
)

const usage = `objdiff - k8s object diff by object id

# Usage

  objdiff [flags] LEFT_FILE RIGHT_FILE

Either LEFT_FILE or RIGHT_FILE can be set to "-". Here, "-" represents stdin.

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
  type: "Diff type (add or change or destroy)"

## markdown

  # Objdiff Summary

  Left file <-> Right file

  | **add** | **change** | **destroy** |
  | :---: | :---: | :---: |
  | x | y | z |
  ## Diff type Object ID

  <details><summary>View Diff</summary>
  Unified diff
  </details>

or

  # Objdiff Summary

  Left file <-> Right file

  No changes.

# Exit status

0 if inputs are the same.
1 if inputs differ.
Otherwise 2.

# Override differ

  objdiff -x diff left.yml right.yml
invokes
  diff --unified=3 --color=never --label left.yml --label right.yml LEFT_FILE RIGHT_FILE

  OBJDIFF_DIFF_CMD='diff' objdiff -c -C 5 left.yml right.yml
invokes
  diff --unified=5 --color=always --label left.yml --label right.yml LEFT_FILE RIGHT_FILE

# Configuration & Precedence

Configuration values are resolved in the following order of precedence (highest to lowest):
1. Command-line flags (e.g. --diff-cmd)
2. Environment variables prefixed with OBJDIFF_ (e.g. OBJDIFF_DIFF_CMD, OBJDIFF_CONTEXT)
3. Default values defined for each flag

Environment variables are derived from flag names in uppercase with hyphens replaced by underscores.
e.g. --ignore-matching-lines -> OBJDIFF_IGNORE_MATCHING_LINES

# Flags`

const (
	exitCodeDiffFound = 1
	exitCodeFailure   = 2
)

func main() {
	fs := pflag.NewFlagSet("main", pflag.ContinueOnError)
	fs.Usage = func() {
		fmt.Printf("%s\n\n", usage)
		fs.PrintDefaults()
	}

	fs.Bool("version", false, "print objdiff version")

	c, err := structconfig.NewConfigWithMerge(
		structconfig.New[config.Config](),
		structconfig.NewMerger[config.Config](),
		fs,
		structconfig.WithEnvPrefix("OBJDIFF_"),
		structconfig.WithArguments(os.Args[1:]),
	)
	if errors.Is(err, pflag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(exitCodeFailure)
	}
	c.Stdin = os.Stdin

	setupLogger(os.Stderr, c.Debug, c.Quiet)

	if v, _ := fs.GetBool("version"); v {
		version.Write(os.Stdout)
		return
	}
	if c.Context < 0 {
		slog.Error("invalid context length")
		os.Exit(exitCodeFailure)
	}

	if fs.NArg() != 2 {
		slog.Error("2 files are required")
		os.Exit(exitCodeFailure)
	}
	if c.OutMode() == config.OutModeUnknown {
		slog.Error("invalid out", slog.String("out", c.Out))
		os.Exit(exitCodeFailure)
	}

	if err := c.Run(os.Stdout, fs.Arg(0), fs.Arg(1)); err != nil {
		if errors.Is(err, config.ErrDiffFound) {
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
