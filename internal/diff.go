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

type DiffRequest struct {
	LeftLabel  string
	RightLabel string
	Left       string
	Right      string
	Color      bool
	Context    int
}

type DiffResponse struct {
	Diff string
}

type Differ interface {
	Diff(ctx context.Context, req *DiffRequest) (*DiffResponse, error)
}

var _ Differ = &ProcessDiffer{}

type ProcessDiffer struct {
	command string
	args    []string
}

func NewProcessDiffer(command string, args []string) *ProcessDiffer {
	return &ProcessDiffer{
		command: command,
		args:    args,
	}
}

func (d *ProcessDiffer) generateArgs(req *DiffRequest, srcFile, destFile string) []string {
	// options from diffutils diff
	args := []string{
		fmt.Sprintf("--unified=%d", req.Context),
	}
	if req.Color {
		args = append(args, "--color=always")
	} else {
		args = append(args, "--color=never")
	}
	args = append(args, "--label", req.LeftLabel, "--label", req.RightLabel)
	args = append(args, d.args...)
	args = append(args, srcFile, destFile)
	return args
}

func (d *ProcessDiffer) Diff(ctx context.Context, req *DiffRequest) (*DiffResponse, error) {
	if req.Left == req.Right {
		return &DiffResponse{}, nil
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
		leftFile  = filepath.Join(dir, "left.txt")
		rightFile = filepath.Join(dir, "right.txt")
	)
	if err := writeFile(leftFile, req.Left); err != nil {
		return nil, fmt.Errorf("failed to get diff: write left file: %w", err)
	}
	if err := writeFile(rightFile, req.Right); err != nil {
		return nil, fmt.Errorf("failed to get diff: write right file: %w", err)
	}

	cmd := exec.CommandContext(ctx, d.command, d.generateArgs(req, leftFile, rightFile)...)
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("invoke command to get diff", slog.String("command", fmt.Sprintf("%#v", cmd.Args)))
	err = cmd.Run()
	if err == nil { // nodiff, exit with 0
		return &DiffResponse{
			Diff: stdout.String(),
		}, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		// inputs are different
		return &DiffResponse{
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

type DMPDiffer struct{}

func NewDMPDiffer() *DMPDiffer {
	return &DMPDiffer{}
}

var _ Differ = &DMPDiffer{}

func (*DMPDiffer) Diff(ctx context.Context, req *DiffRequest) (*DiffResponse, error) {
	dmp := &DMP{
		LeftLabel:  req.LeftLabel,
		RightLabel: req.RightLabel,
		Context:    req.Context,
	}
	diff, err := dmp.Diff(req.Left, req.Right)
	if err != nil {
		if errors.Is(err, ErrDMPNoDiff) {
			return &DiffResponse{}, nil
		}
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	return &DiffResponse{
		Diff: diff.IntoString(req.Color),
	}, nil
}

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

var _ ObjectDiffer = &ObjectDiffBuilder{}

func NewObjectDiffBuilder(
	differ Differ,
	left, right string,
	diffContext int,
	color bool,

) *ObjectDiffBuilder {
	return &ObjectDiffBuilder{
		differ:      differ,
		left:        left,
		right:       right,
		color:       color,
		diffContext: diffContext,
	}
}

type ObjectDiffBuilder struct {
	differ      Differ
	left        string
	right       string
	diffContext int
	color       bool
}

func (d *ObjectDiffBuilder) ObjectDiff(ctx context.Context, pair *ObjectPair) (*ObjectDiff, error) {
	var leftBody, rightBody string
	if x := pair.Left; x != nil {
		leftBody = x.Body
	}
	if x := pair.Right; x != nil {
		rightBody = x.Body
	}

	diff, err := d.differ.Diff(ctx, &DiffRequest{
		Left:       leftBody,
		Right:      rightBody,
		LeftLabel:  newDiffHeader(d.left, pair.ID, d.color),
		RightLabel: newDiffHeader(d.right, pair.ID, d.color),
		Color:      d.color,
		Context:    d.diffContext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: id=%s: %w", pair.ID, err)
	}

	if diff.Diff == "" {
		return &ObjectDiff{
			Pair: pair,
		}, nil
	}

	return &ObjectDiff{
		Pair: pair,
		Diff: diff.Diff,
	}, nil
}
