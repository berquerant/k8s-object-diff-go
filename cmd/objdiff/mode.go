package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

type diffPrinter struct {
	mode      outMode
	pairs     []*internal.ObjectPair
	differ    internal.ObjectDiffer
	marshaler internal.Marshaler
}

func (p *diffPrinter) print(ctx context.Context) error {
	switch p.mode {
	case outModeID:
		return p.printObjectIDList()
	case outModeYaml:
		return p.printYamlDiff(ctx)
	default:
		return p.printTextDiff()
	}
}

func (p *diffPrinter) printObjectIDList() error {
	xs := make([]string, len(p.pairs))
	for i, x := range p.pairs {
		xs[i] = x.ID
	}
	fmt.Println(strings.Join(xs, "\n"))
	return nil
}

func (p *diffPrinter) printTextDiff() error {
	var diffFound bool

	for _, x := range p.pairs {
		slog.Debug("process pair", slog.String("id", x.ID))
		if x.IsMissing() {
			slog.Error("missing object", slog.String("id", x.ID))
			continue
		}
		d := p.differ.ObjectDiff(x)
		if d.Diff == "" {
			slog.Debug("no diff", slog.String("id", x.ID))
			continue
		}
		if !diffFound {
			diffFound = true
		}
		fmt.Print(d.Diff)
	}

	if diffFound {
		return errDiffFound
	}
	return nil
}

func (p *diffPrinter) printYamlDiff(ctx context.Context) error {
	var (
		diffFound bool
		result    []any
	)

	for _, x := range p.pairs {
		slog.Debug("process pair", slog.String("id", x.ID))
		if x.IsMissing() {
			slog.Error("missing object", slog.String("id", x.ID))
			continue
		}
		d := p.differ.ObjectDiff(x)
		if d.Diff == "" {
			slog.Debug("no diff", slog.String("id", x.ID))
			continue
		}
		if !diffFound {
			diffFound = true
		}
		y := map[string]any{
			"id":   d.Pair.ID,
			"diff": d.Diff,
		}
		if a := d.Pair.Left; a != nil {
			y["left"] = a.Body
		}
		if a := d.Pair.Right; a != nil {
			y["right"] = a.Body
		}
		result = append(result, y)
	}

	if len(result) == 0 {
		return nil
	}

	b, err := p.marshaler.Marshal(ctx, result)
	if err != nil {
		return err
	}
	fmt.Print(string(b))

	if diffFound {
		return errDiffFound
	}
	return nil
}
