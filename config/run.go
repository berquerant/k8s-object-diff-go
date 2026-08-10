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
	if left == right && left == stdinFilename {
		return fmt.Errorf("'%s' cannot be specified for both left and right", stdinFilename)
	}

	lineFilter, err := internal.NewLineFilter(c.IgnoreMatchingLines)
	if err != nil {
		return fmt.Errorf("ignore-matching-lines: %w", err)
	}

	var yqFilters []*internal.YqFilter
	if f := internal.NewFieldFilter(c.IgnoreFields); f != nil {
		yqFilters = append(yqFilters, f)
	}
	if f := internal.NewLabelFilter(c.IgnoreLabels); f != nil {
		yqFilters = append(yqFilters, f)
	}
	if f := internal.NewAnnotationFilter(c.IgnoreAnnotations); f != nil {
		yqFilters = append(yqFilters, f)
	}
	if c.IgnoreManagedFields {
		if f := internal.NewFieldFilter([]string{"metadata.managedFields"}); f != nil {
			yqFilters = append(yqFilters, f)
		}
	}
	if c.IgnoreStatus {
		if f := internal.NewFieldFilter([]string{"status"}); f != nil {
			yqFilters = append(yqFilters, f)
		}
	}

	loader := newObjectLoader(c, lineFilter, yqFilters)
	leftMap, err := loader.load(ctx, left)
	if err != nil {
		return fmt.Errorf("left file: %s: %w", left, err)
	}
	rightMap, err := loader.load(ctx, right)
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

	switch {
	case len(c.Labels) == 1:
		left = c.Labels[0]
	case len(c.Labels) > 1:
		left, right = c.Labels[0], c.Labels[1]
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
		marshaler:            internal.NewYamlMarshaler(c.Indent, false),
		color:                c.Color,
		diffContext:          c.Context,
		left:                 left,
		right:                right,
		out:                  w,
		verbose:              c.Verbose,
		markdownHeadingLevel: c.MarkdownHeadingLevel,
	}

	return printer.print(ctx)
}

const (
	stdinFilename = "-"
)

type objectLoader struct {
	marshaler            internal.Marshaler
	stdin                io.Reader
	sep                  string
	allowDuplicateMapKey bool
	lineFilter           *internal.LineFilter
	yqFilters            []*internal.YqFilter
}

func newObjectLoader(c *Config, lineFilter *internal.LineFilter, yqFilters []*internal.YqFilter) *objectLoader {
	return &objectLoader{
		marshaler:            internal.NewYamlMarshaler(c.Indent, true),
		stdin:                c.Stdin,
		sep:                  c.Separator,
		allowDuplicateMapKey: c.AllowDuplicateKey,
		lineFilter:           lineFilter,
		yqFilters:            yqFilters,
	}
}

func (l *objectLoader) load(ctx context.Context, file string) (*internal.ObjectMap, error) {
	slog.Debug("loadObjects", slog.String("file", file))

	var r io.Reader
	switch file {
	case stdinFilename:
		r = l.stdin
	default:
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", file, err)
		}
		defer func() {
			_ = f.Close()
		}()
		r = f
	}

	objects, err := internal.LoadObjects(ctx, r, l.marshaler, l.allowDuplicateMapKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load objects from %s: %w", file, err)
	}
	slog.Debug("loaded objects", slog.String("file", file), slog.Int("len", len(objects)))

	objectMap := internal.NewObjectMap(l.sep)
	for _, x := range objects {
		x.Body = l.lineFilter.Filter(x.Body)
		for _, f := range l.yqFilters {
			filteredBody, err := f.FilterBody(x.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to filter body in %s: %w", file, err)
			}
			x.Body = filteredBody
		}

		slog.Debug("add object", slog.String("file", file), slog.String("id", x.Header.IntoID(l.sep)))
		if objectMap.Add(x) {
			slog.Warn("duplicated object",
				slog.String("id", x.Header.IntoID(l.sep)),
				slog.String("file", file),
			)
		}
	}

	return objectMap, nil
}
