package internal

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type ObjectDiff struct {
	Pair *ObjectPair
	Diff string
}

type ObjectDiffer interface {
	ObjectDiff(pair *ObjectPair) *ObjectDiff
}

var _ ObjectDiffer = &ObjectDiffBuilder{}

type ObjectDiffBuilder struct {
	color       bool
	diffContext int
	left        string
	right       string
}

func NewObjectDiffBuilder(left, right string, diffContext int, color bool) *ObjectDiffBuilder {
	return &ObjectDiffBuilder{
		color:       color,
		diffContext: diffContext,
		left:        left,
		right:       right,
	}
}

func (d *ObjectDiffBuilder) ObjectDiff(pair *ObjectPair) *ObjectDiff {
	var leftBody, rightBody string
	if x := pair.Left; x != nil {
		leftBody = x.Body
	}
	if x := pair.Right; x != nil {
		rightBody = x.Body
	}

	if leftBody == rightBody {
		return &ObjectDiff{
			Pair: pair,
		}
	}

	u := difflib.UnifiedDiff{
		A:        difflib.SplitLines(leftBody),
		B:        difflib.SplitLines(rightBody),
		FromFile: fmt.Sprintf("%s %s", d.left, pair.ID),
		ToFile:   fmt.Sprintf("%s %s", d.right, pair.ID),
		Context:  d.diffContext,
	}
	if d.color {
		u.FromFile = NewDiffHeader(u.FromFile)
		u.ToFile = NewDiffHeader(u.ToFile)
	}
	diff, _ := difflib.GetUnifiedDiffString(u)
	if d.color {
		diffs := strings.Split(diff, "\n")
		for i, x := range diffs {
			if len(x) == 0 {
				continue
			}
			switch x[0] {
			case '-':
				diffs[i] = NewDeleteString(x)
			case '+':
				diffs[i] = NewInsertString(x)
			}
		}
		diff = strings.Join(diffs, "\n")
	}

	return &ObjectDiff{
		Pair: pair,
		Diff: diff,
	}
}

func NewDiffHeader(s string) string {
	// yellow
	return fmt.Sprintf("\x1b[33m%s\x1b[0m", s)
}

func NewDeleteString(s string) string {
	// red
	return fmt.Sprintf("\x1b[31m%s\x1b[0m", s)
}

func NewInsertString(s string) string {
	// green
	return fmt.Sprintf("\x1b[32m%s\x1b[0m", s)
}
