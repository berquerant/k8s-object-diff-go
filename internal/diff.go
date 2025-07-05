package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type ObjectDiff struct {
	Pair *ObjectPair
	Diff string
}

type ObjectDiffer interface {
	ObjectDiff(ctx context.Context, pair *ObjectPair) (*ObjectDiff, error)
}

func newDiffHeader(name, objectID string, color bool) string {
	x := name + " " + objectID
	if color {
		return yellowString(x)
	}
	return x
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

	if d.color {
		args = append(args, "--color=always")
	} else {
		args = append(args, "--color=never")
	}

	args = append(args, "--label", newDiffHeader(d.left, objectID, d.color), "--label", newDiffHeader(d.right, objectID, d.color))
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

	dmp := &DMP{
		LeftLabel:  newDiffHeader(d.left, pair.ID, d.color),
		RightLabel: newDiffHeader(d.right, pair.ID, d.color),
		Context:    d.diffContext,
	}
	diff, err := dmp.Diff(leftBody, rightBody)
	if err != nil {
		if errors.Is(err, ErrDMPNoDiff) {
			return &ObjectDiff{
				Pair: pair,
			}, nil
		}
		return nil, fmt.Errorf("failed to get diff: id=%s: %w", pair.ID, err)
	}

	return &ObjectDiff{
		Pair: pair,
		Diff: diff.IntoString(d.color),
	}, nil
}
