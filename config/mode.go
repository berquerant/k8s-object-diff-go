package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

type diffPrinter struct {
	mode                 OutMode
	pairs                []*internal.ObjectPair
	differ               internal.Differ
	objectDiffer         internal.ObjectDiffer
	marshaler            internal.Marshaler
	diffContext          int
	color                bool
	left                 string
	right                string
	out                  io.Writer
	verbose              bool
	markdownHeadingLevel uint
}

func (p *diffPrinter) print(ctx context.Context) error {
	switch p.mode {
	case OutModeID:
		return p.printObjectIDDiff(ctx)
	case OutModeIDList:
		return p.printObjectIDList()
	case OutModeYaml:
		return p.printYamlDiff(ctx)
	case OutModeMarkdown:
		return p.printMarkdownDiff(ctx)
	default:
		return p.printTextDiff(ctx)
	}
}

func (diffPrinter) describeDiffType(diffType internal.DiffType) string {
	switch diffType {
	case internal.DiffTypeUnchange:
		return "no changes"
	case internal.DiffTypeAdd:
		return "created"
	case internal.DiffTypeChange:
		return "updated"
	case internal.DiffTypeDestroy:
		return "destroyed"
	default:
		return "unknown"
	}
}

func (p *diffPrinter) diffTypeString(id string, diffType internal.DiffType) string {
	id = fmt.Sprintf("# %s", id)
	desc := p.describeDiffType(diffType)
	if p.color {
		id = internal.BoldString(id)
		if diffType == internal.DiffTypeDestroy {
			desc = internal.RedString(desc)
		}
	}
	return fmt.Sprintf("%s will be %s", id, desc)
}

func (p *diffPrinter) diffTypeSummary(add, change, destroy int) string {
	head := "Summary:"
	if p.color {
		head = internal.BoldString(head)
	}
	return fmt.Sprintf("%s %d to %s, %d to %s, %d to %s.",
		head,
		add, internal.DiffTypeAdd,
		change, internal.DiffTypeChange,
		destroy, internal.DiffTypeDestroy,
	)
}

func (p *diffPrinter) printObjectIDList() error {
	xs := make([]string, len(p.pairs))
	for i, x := range p.pairs {
		xs[i] = x.ID
	}
	_, _ = fmt.Fprintln(p.out, strings.Join(xs, "\n"))
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
	_, _ = fmt.Fprint(p.out, d.Diff)
	return ErrDiffFound
}

func (p *diffPrinter) printTextDiff(ctx context.Context) error {
	var (
		diffFound            bool
		add, change, destroy int
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
		switch d.Type {
		case internal.DiffTypeAdd:
			add++
		case internal.DiffTypeChange:
			change++
		case internal.DiffTypeDestroy:
			destroy++
		}
		if p.verbose {
			_, _ = fmt.Fprintln(p.out, p.diffTypeString(x.ID, d.Type))
		}
		_, _ = fmt.Fprint(p.out, d.Diff)
	}
	if p.verbose {
		_, _ = fmt.Fprintf(p.out, "\n%s\n", p.diffTypeSummary(add, change, destroy))
	}

	if diffFound {
		return ErrDiffFound
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
			"type": d.Type.String(),
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
	_, _ = fmt.Fprint(p.out, string(b))

	if diffFound {
		return ErrDiffFound
	}
	return nil
}

func (p *diffPrinter) printMarkdownDiff(ctx context.Context) error {
	var (
		heading = func(n int) string {
			return strings.Repeat("#", n+int(p.markdownHeadingLevel))
		}
		diffTypeAsString = func(x internal.DiffType) string {
			switch x {
			case internal.DiffTypeAdd:
				return "add"
			case internal.DiffTypeChange:
				return "change"
			case internal.DiffTypeDestroy:
				return "destroy"
			default:
				return "unknown"
			}
		}
		summaryNoDiff = fmt.Sprintf(`%s Objdiff Summary

%s <-> %s

No changes.`,
			heading(0),
			p.left,
			p.right,
		)
		summaryTmpl = fmt.Sprintf(`%s Objdiff Summary

%s <-> %s

| **%s** | **%s** | **%s** |
| :---: | :---: | :---: |
| {{ .Add }} | {{ .Change }} | {{ .Destroy }} |`,
			heading(0),
			"`{{ .Left }}`",
			"`{{ .Right }}`",
			diffTypeAsString(internal.DiffTypeAdd),
			diffTypeAsString(internal.DiffTypeChange),
			diffTypeAsString(internal.DiffTypeDestroy),
		)
		diffTmpl = fmt.Sprintf(`{{ range . }}
%s {{ .DiffType }} %s

<details><summary>View Diff</summary>

%s

</details>
{{ end }}`,
			heading(1),
			"`{{ .ID }}`",
			"``` diff\n{{ .Diff }}```",
		)
	)

	type Summary struct {
		Left, Right          string
		Add, Change, Destroy int
	}
	type Item struct {
		ID, Diff, DiffType string
	}

	var (
		diffFound bool
		summary   = Summary{
			Left:  p.left,
			Right: p.right,
		}
		items []Item
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
		switch d.Type {
		case internal.DiffTypeAdd:
			summary.Add++
		case internal.DiffTypeChange:
			summary.Change++
		case internal.DiffTypeDestroy:
			summary.Destroy++
		}
		items = append(items, Item{
			ID:       x.ID,
			Diff:     d.Diff,
			DiffType: diffTypeAsString(d.Type),
		})
	}

	if !diffFound {
		fmt.Println(summaryNoDiff)
		return nil
	}

	if err := template.Must(template.New("summary").Parse(summaryTmpl)).Execute(os.Stdout, summary); err != nil {
		return err
	}
	if err := template.Must(template.New("diff").Parse(diffTmpl)).Execute(os.Stdout, items); err != nil {
		return err
	}
	return ErrDiffFound
}
