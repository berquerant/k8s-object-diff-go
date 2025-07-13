package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

var ErrDiffFound = errors.New("DiffFound")

func (c *Config) Run(w io.Writer, left, right string) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()
	return c.runObjDiff(ctx, w, left, right)
}

func (c *Config) runObjDiff(ctx context.Context, w io.Writer, left, right string) error {
	marshaler := internal.NewYamlMarshaler(c.Indent, true)
	leftMap, err := loadObjects(ctx, marshaler, left, c.Separator, c.AllowDuplicateKey)
	if err != nil {
		return fmt.Errorf("left file: %s: %w", left, err)
	}
	rightMap, err := loadObjects(ctx, marshaler, right, c.Separator, c.AllowDuplicateKey)
	if err != nil {
		return fmt.Errorf("right file: %s: %w", right, err)
	}

	pairMap := internal.NewObjectPairMap(leftMap, rightMap)
	pairs := pairMap.ObjectPairs()
	slog.Debug("found pairs", slog.Int("len", len(pairs)))

	differ, err := c.newDiffer()
	if err != nil {
		return fmt.Errorf("differ: %w", err)
	}

	printer := &diffPrinter{
		mode:   c.OutMode(),
		pairs:  pairs,
		differ: differ,
		objectDiffer: internal.NewObjectDiffBuilder(
			differ,
			left, right,
			c.Context,
			c.Color,
		),
		marshaler:   internal.NewYamlMarshaler(c.Indent, false),
		color:       c.Color,
		diffContext: c.Context,
		left:        left,
		right:       right,
		out:         w,
	}

	return printer.print(ctx)
}

func loadObjects(ctx context.Context, marshaler internal.Marshaler, file, sep string, allowDuplicateMapKey bool) (*internal.ObjectMap, error) {
	slog.Debug("loadObjects", slog.String("file", file))
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", file, err)
	}
	defer func() {
		_ = f.Close()
	}()

	objects, err := internal.LoadObjects(ctx, f, marshaler, allowDuplicateMapKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load objects from %s: %w", file, err)
	}
	slog.Debug("loaded objects", slog.String("file", file), slog.Int("len", len(objects)))

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
