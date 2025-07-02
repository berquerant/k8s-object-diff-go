package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

var errDiffFound = errors.New("DiffFound")

func run(c *Config, left, right string) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()
	return runObjDiff(ctx, c, left, right)
}

func runObjDiff(ctx context.Context, c *Config, left, right string) error {
	marshaler := internal.NewYamlMarshaler(c.Indent, true)
	leftMap, err := loadObjects(ctx, marshaler, left, c.Separator, c.AllowDuplicateKey)
	if err != nil {
		return fmt.Errorf("%w: left file: %s", err, left)
	}
	rightMap, err := loadObjects(ctx, marshaler, right, c.Separator, c.AllowDuplicateKey)
	if err != nil {
		return fmt.Errorf("%w: right file: %s", err, right)
	}

	pairMap := internal.NewObjectPairMap(leftMap, rightMap)
	pairs := pairMap.ObjectPairs()
	slog.Debug("found pairs", slog.Int("len", len(pairs)))

	printer := &diffPrinter{
		mode:      c.outMode(),
		pairs:     pairs,
		differ:    internal.NewObjectDiffBuilder(left, right, c.Context, c.Color),
		marshaler: internal.NewYamlMarshaler(c.Indent, false),
	}

	return printer.print(ctx)
}

func loadObjects(ctx context.Context, marshaler internal.Marshaler, file, sep string, allowDuplicateMapKey bool) (*internal.ObjectMap, error) {
	slog.Debug("loadObjects", slog.String("file", file))
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	objects, err := internal.LoadObjects(ctx, f, marshaler, allowDuplicateMapKey)
	if err != nil {
		return nil, err
	}

	objectMap := internal.NewObjectMap(sep)
	for _, x := range objects {
		slog.Debug("add object", slog.String("file", file), slog.String("id", x.Header.IntoID(sep)))
		if objectMap.Add(x) {
			slog.Warn("duplicated object",
				slog.String("id", x.Header.IntoID(sep)),
				slog.String("file", file),
			)
		}
	}

	return objectMap, nil
}
