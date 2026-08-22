package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/config"
	"github.com/berquerant/k8s-object-diff-go/version"
	"github.com/berquerant/structconfig"
	"github.com/spf13/pflag"
)

const (
	exitCodeDiffFound = 1
	exitCodeFailure   = 2
)

func main() {
	fs := pflag.NewFlagSet("main", pflag.ContinueOnError)
	fs.Usage = func() {
		flagsUsage := strings.TrimRight(fs.FlagUsages(), "\n")
		fmt.Printf("%s\n```\n%s\n```\n", config.Usage(), flagsUsage)
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
