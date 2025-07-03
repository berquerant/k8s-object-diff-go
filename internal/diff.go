package internal

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type ObjectDiff struct {
	Pair *ObjectPair
	Diff string
}

type ObjectDiffer interface {
	ObjectDiff(ctx context.Context, pair *ObjectPair) (*ObjectDiff, error)
}

type ProcessObjectDiffBuilder struct {
	command     string
	args        []string
	left        string
	right       string
	color       bool
	diffContext int
}

var _ ObjectDiffer = &ProcessObjectDiffBuilder{}

func NewProcessObjectDiffBuilder(
	command string,
	args []string,
	left, right string,
	diffContext int,
	color bool,
) *ProcessObjectDiffBuilder {
	return &ProcessObjectDiffBuilder{
		command:     command,
		args:        args,
		left:        left,
		right:       right,
		color:       color,
		diffContext: diffContext,
	}
}

func (d *ProcessObjectDiffBuilder) generateArgs(objectID, srcFile, destFile string) []string {
	// options from diffutils diff
	args := []string{
		fmt.Sprintf("--unified=%d", d.diffContext),
	}

	left := d.left + " " + objectID
	right := d.right + " " + objectID
	if d.color {
		left = NewDiffHeader(left)
		right = NewDiffHeader(right)
		args = append(args, "--color=always")
	} else {
		args = append(args, "--color=never")
	}

	args = append(args, "--label", left, "--label", right)
	args = append(args, d.args...)
	args = append(args, srcFile, destFile)
	return args
}

func (d *ProcessObjectDiffBuilder) ObjectDiff(ctx context.Context, pair *ObjectPair) (*ObjectDiff, error) {
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
		}, nil
	}

	dir, err := os.MkdirTemp("", "objdiff")
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: mkdir: %w", err)
	}
	defer os.RemoveAll(dir)

	writeFile := func(name, content string) error {
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprint(f, content)
		return err
	}
	var (
		leftFile  = filepath.Join(dir, "left.yml")
		rightFile = filepath.Join(dir, "right.yml")
	)
	if err := writeFile(leftFile, leftBody); err != nil {
		return nil, fmt.Errorf("failed to get diff: write left file: %w", err)
	}
	if err := writeFile(rightFile, rightBody); err != nil {
		return nil, fmt.Errorf("failed to get diff: write right file: %w", err)
	}

	cmd := exec.CommandContext(ctx, d.command, d.generateArgs(pair.ID, leftFile, rightFile)...)
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("invoke command to get diff", slog.String("command", fmt.Sprintf("%#v", cmd.Args)))
	err = cmd.Run()
	if err == nil { // nodiff, exit with 0
		return &ObjectDiff{
			Pair: pair,
			Diff: stdout.String(),
		}, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		// inputs are different
		return &ObjectDiff{
			Pair: pair,
			Diff: stdout.String(),
		}, nil
	}

	// trouble
	return nil, fmt.Errorf("failed to get diff: invoke %#v: stderr=%s: %w",
		cmd.Args,
		stderr.String(),
		err,
	)
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

func (d *ObjectDiffBuilder) ObjectDiff(_ context.Context, pair *ObjectPair) (*ObjectDiff, error) {
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
		}, nil
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
	diff, err := difflib.GetUnifiedDiffString(u)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: id=%s: %w", pair.ID, err)
	}
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
	}, nil
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
