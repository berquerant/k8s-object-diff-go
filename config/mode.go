package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

// collectDiffs iterates over all pairs, calls ObjectDiff on each, and returns
// only the pairs that have a non-empty diff. Missing pairs are logged and skipped.
func (p *diffPrinter) collectDiffs(ctx context.Context) ([]*internal.ObjectDiff, error) {
	var result []*internal.ObjectDiff
	for _, x := range p.pairs {
		slog.Debug("process pair", slog.String("id", x.ID))
		if x.IsMissing() {
			slog.Error("missing object", slog.String("id", x.ID))
			continue
		}
		d, err := p.objectDiffer.ObjectDiff(ctx, x)
		if err != nil {
			return nil, err
		}
		if d.Diff == "" {
			slog.Debug("no diff", slog.String("id", x.ID))
			continue
		}
		result = append(result, d)
	}
	return result, nil
}

// diffStats holds counts of each diff type.
type diffStats struct {
	Add, Change, Destroy int
}

// countDiffStats aggregates add/change/destroy counts from a slice of ObjectDiffs.
func countDiffStats(diffs []*internal.ObjectDiff) diffStats {
	var s diffStats
	for _, d := range diffs {
		switch d.Type {
		case internal.DiffTypeAdd:
			s.Add++
		case internal.DiffTypeChange:
			s.Change++
		case internal.DiffTypeDestroy:
			s.Destroy++
		}
	}
	return s
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
	diffs, err := p.collectDiffs(ctx)
	if err != nil {
		return err
	}
	stats := countDiffStats(diffs)
	for _, d := range diffs {
		if p.verbose {
			_, _ = fmt.Fprintln(p.out, p.diffTypeString(d.Pair.ID, d.Type))
		}
		_, _ = fmt.Fprint(p.out, d.Diff)
	}
	if p.verbose {
		_, _ = fmt.Fprintf(p.out, "\n%s\n", p.diffTypeSummary(stats.Add, stats.Change, stats.Destroy))
	}
	if len(diffs) == 0 {
		return nil
	}
	return ErrDiffFound
}

func (p *diffPrinter) printYamlDiff(ctx context.Context) error {
	diffs, err := p.collectDiffs(ctx)
	if err != nil {
		return err
	}
	if len(diffs) == 0 {
		return nil
	}

	result := make([]any, 0, len(diffs))
	for _, d := range diffs {
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

	b, err := p.marshaler.Marshal(ctx, result)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(p.out, string(b))
	return ErrDiffFound
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

	diffs, err := p.collectDiffs(ctx)
	if err != nil {
		return err
	}
	if len(diffs) == 0 {
		_, _ = fmt.Fprintln(p.out, summaryNoDiff)
		return nil
	}

	stats := countDiffStats(diffs)
	summary := Summary{
		Left:    p.left,
		Right:   p.right,
		Add:     stats.Add,
		Change:  stats.Change,
		Destroy: stats.Destroy,
	}
	items := make([]Item, 0, len(diffs))
	for _, d := range diffs {
		items = append(items, Item{
			ID:       d.Pair.ID,
			Diff:     d.Diff,
			DiffType: diffTypeAsString(d.Type),
		})
	}

	if err := template.Must(template.New("summary").Parse(summaryTmpl)).Execute(p.out, summary); err != nil {
		return err
	}
	if err := template.Must(template.New("diff").Parse(diffTmpl)).Execute(p.out, items); err != nil {
		return err
	}
	return ErrDiffFound
}
