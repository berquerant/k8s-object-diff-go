package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

type diffPrinter struct {
	mode         outMode
	pairs        []*internal.ObjectPair
	differ       internal.Differ
	objectDiffer internal.ObjectDiffer
	marshaler    internal.Marshaler
	diffContext  int
	color        bool
	left         string
	right        string
}

func (p *diffPrinter) print(ctx context.Context) error {
	switch p.mode {
	case outModeID:
		return p.printObjectIDDiff(ctx)
	case outModeIDList:
		return p.printObjectIDList()
	case outModeYaml:
		return p.printYamlDiff(ctx)
	default:
		return p.printTextDiff(ctx)
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

func (p *diffPrinter) printObjectIDDiff(ctx context.Context) error {
	var (
		leftIDList  []string
		rightIDList []string
	)
	for _, x := range p.pairs {
		if x.Left != nil {
			leftIDList = append(leftIDList, x.ID)
		}
		if x.Right != nil {
			rightIDList = append(rightIDList, x.ID)
		}
	}
	var (
		join = func(xs []string) string {
			v := strings.Join(xs, "\n")
			if v != "" {
				return v + "\n"
			}
			return ""
		}
		left      = join(leftIDList)
		right     = join(rightIDList)
		newHeader = func(s string) string {
			if p.color {
				return internal.YellowString(s)
			}
			return s
		}
	)
	d, err := p.differ.Diff(ctx, &internal.DiffRequest{
		Left:       left,
		Right:      right,
		LeftLabel:  newHeader(p.left),
		RightLabel: newHeader(p.right),
		Color:      p.color,
		Context:    p.diffContext,
	})
	if err != nil {
		return err
	}
	if d.Diff == "" {
		slog.Debug("no diff")
		return nil
	}
	fmt.Print(d.Diff)
	return errDiffFound
}

func (p *diffPrinter) printTextDiff(ctx context.Context) error {
	var diffFound bool

	for _, x := range p.pairs {
		slog.Debug("process pair", slog.String("id", x.ID))
		if x.IsMissing() {
			slog.Error("missing object", slog.String("id", x.ID))
			continue
		}
		d, err := p.objectDiffer.ObjectDiff(ctx, x)
		if err != nil {
			return err
		}
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
		d, err := p.objectDiffer.ObjectDiff(ctx, x)
		if err != nil {
			return err
		}
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
