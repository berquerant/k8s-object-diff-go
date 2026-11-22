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
	if left == stdinFilename && right == stdinFilename {
		return fmt.Errorf("'%s' cannot be specified for both left and right", stdinFilename)
	}

	leftReader, leftClose, err := openFileOrStdin(left, c.Stdin)
	if err != nil {
		return fmt.Errorf("left file: %s: %w", left, err)
	}
	defer leftClose()

	rightReader, rightClose, err := openFileOrStdin(right, c.Stdin)
	if err != nil {
		return fmt.Errorf("right file: %s: %w", right, err)
	}
	defer rightClose()

	left, right = c.resolveLabels(left, right)
	return c.runWithReadersAndLabels(ctx, w, leftReader, rightReader, left, right)
}

func openFileOrStdin(file string, stdin io.Reader) (io.Reader, func(), error) {
	switch file {
	case stdinFilename:
		return stdin, func() {}, nil
	default:
		f, err := os.Open(file) //nolint:gosec // CLI intentionally opens user-specified manifest files
		if err != nil {
			return nil, nil, err
		}
		return f, func() { _ = f.Close() }, nil
	}
}

func (c *Config) resolveLabels(defaultLeft, defaultRight string) (string, string) {
	switch {
	case len(c.Labels) == 1:
		return c.Labels[0], defaultRight
	case len(c.Labels) > 1:
		return c.Labels[0], c.Labels[1]
	default:
		return defaultLeft, defaultRight
	}
}

func (c *Config) buildYqFilters() ([]*internal.YqFilter, error) {
	var yqFilters []*internal.YqFilter
	appendFilter := func(f *internal.YqFilter) {
		if f != nil && !f.IsEmpty() {
			yqFilters = append(yqFilters, f)
		}
	}
	fieldFilter, err := internal.NewFieldFilter(c.IgnoreFields)
	if err != nil {
		return nil, fmt.Errorf("ignore-field: %w", err)
	}
	appendFilter(fieldFilter)

	labelFilter, err := internal.NewLabelFilter(c.IgnoreLabels)
	if err != nil {
		return nil, fmt.Errorf("ignore-label: %w", err)
	}
	appendFilter(labelFilter)

	annotationFilter, err := internal.NewAnnotationFilter(c.IgnoreAnnotations)
	if err != nil {
		return nil, fmt.Errorf("ignore-annotation: %w", err)
	}
	appendFilter(annotationFilter)

	if c.IgnoreManagedFields {
		f, err := internal.NewFieldFilter([]string{"metadata.managedFields"})
		if err != nil {
			return nil, fmt.Errorf("ignore-managed-fields: %w", err)
		}
		appendFilter(f)
	}
	if c.IgnoreStatus {
		f, err := internal.NewFieldFilter([]string{"status"})
		if err != nil {
			return nil, fmt.Errorf("ignore-status: %w", err)
		}
		appendFilter(f)
	}
	return yqFilters, nil
}

func (c *Config) runWithReaders(ctx context.Context, w io.Writer, leftReader, rightReader io.Reader) error {
	left, right := c.resolveLabels("left", "right")
	return c.runWithReadersAndLabels(ctx, w, leftReader, rightReader, left, right)
}

func (c *Config) runWithReadersAndLabels(ctx context.Context, w io.Writer, leftReader, rightReader io.Reader, left, right string) error {
	if err := c.Validate(); err != nil {
		return err
	}

	lineFilter, err := internal.NewLineFilter(c.IgnoreMatchingLines)
	if err != nil {
		return fmt.Errorf("ignore-matching-lines: %w", err)
	}

	yqFilters, err := c.buildYqFilters()
	if err != nil {
		return err
	}

	loader := newObjectLoader(c, lineFilter, yqFilters)
	leftMap, err := loader.loadFromReader(ctx, leftReader, left)
	if err != nil {
		return fmt.Errorf("left file: %s: %w", left, err)
	}
	rightMap, err := loader.loadFromReader(ctx, rightReader, right)
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

	headingLevel := c.MarkdownHeadingLevel
	if headingLevel == 0 {
		headingLevel = 1
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
		markdownHeadingLevel: headingLevel,
		markdownTitle:        c.MarkdownTitle,
		template:             c.Template,
		allowTemplateFile:    !c.MCP,
	}

	return printer.print(ctx)
}

const (
	stdinFilename = "-"
)

type objectLoader struct {
	marshaler            internal.Marshaler
	sep                  string
	allowDuplicateMapKey bool
	lineFilter           *internal.LineFilter
	yqFilters            []*internal.YqFilter
}

func newObjectLoader(c *Config, lineFilter *internal.LineFilter, yqFilters []*internal.YqFilter) *objectLoader {
	return &objectLoader{
		marshaler:            internal.NewYamlMarshaler(c.Indent, true),
		sep:                  c.Separator,
		allowDuplicateMapKey: c.AllowDuplicateKey,
		lineFilter:           lineFilter,
		yqFilters:            yqFilters,
	}
}

func (l *objectLoader) loadFromReader(ctx context.Context, r io.Reader, label string) (*internal.ObjectMap, error) {
	objects, err := internal.LoadObjects(ctx, r, l.marshaler, l.allowDuplicateMapKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load objects from %s: %w", label, err)
	}
	slog.Debug("loaded objects", slog.String("file", label), slog.Int("len", len(objects)))

	objectMap := internal.NewObjectMap(l.sep)
	for _, x := range objects {
		x.Body = l.lineFilter.Filter(x.Body)
		for _, f := range l.yqFilters {
			filteredBody, err := f.FilterBody(x.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to filter body in %s: %w", label, err)
			}
			x.Body = filteredBody
		}

		slog.Debug("add object", slog.String("file", label), slog.String("id", x.Header.IntoID(l.sep)))
		if objectMap.Add(x) {
			slog.Warn("duplicated object",
				slog.String("id", x.Header.IntoID(l.sep)),
				slog.String("file", label),
			)
		}
	}

	return objectMap, nil
}
