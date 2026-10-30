package internal_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

const (
	diffTestCommand           = "./diff_test.sh"
	diffTestAnyString         = "ANY"
	diffTestLeftFile          = "LEFT"
	diffTestRightFile         = "RIGHT"
	diffTestDiffContext       = 3
	diffTestDiffContextString = "3"
	diffTestObjectID          = "OBJECT_ID"

	diffTestEnvExitCode = "EXIT_CODE"
	diffTestExitSuccess = 0
	diffTestExitDiff    = 1
	diffTestExitFailure = 2
)

func verifyProcessObjectDiff(
	t *testing.T,
	args []string,
	color bool,
	pair *internal.ObjectPair,
	exitCode int,
	noDiff, wantErr bool,
	passedArgs []string,
) {
	t.Helper()
	_ = os.Setenv(diffTestEnvExitCode, strconv.Itoa(exitCode))
	defer os.Unsetenv(diffTestEnvExitCode)

	x := internal.NewObjectDiffBuilder(
		internal.NewProcessDiffer(diffTestCommand, args),
		diffTestLeftFile, diffTestRightFile,
		diffTestDiffContext,
		color,
	)
	got, err := x.ObjectDiff(context.TODO(), pair)
	if wantErr {
		assert.NotNil(t, err)
		return
	}
	if !assert.Nil(t, err) {
		return
	}
	assert.Equal(t, pair, got.Pair)
	if noDiff {
		assert.Empty(t, got.Diff)
		return
	}
	passed := strings.Split(got.Diff, " ")
	if !assert.Equal(t, len(passedArgs), len(passed), "want:%#v\ngot:%#v", passedArgs, passed) {
		return
	}
	for i, w := range passedArgs {
		g := passed[i]
		if w == diffTestAnyString {
			continue
		}
		assert.Equal(t, w, g, "passed[%d]", i)
	}
}

func TestProcessObjectDiffBuilder(t *testing.T) {
	noDiffPair := &internal.ObjectPair{
		ID:    diffTestObjectID,
		Left:  &internal.Object{},
		Right: &internal.Object{},
	}
	diffPair := &internal.ObjectPair{
		ID:    diffTestObjectID,
		Left:  &internal.Object{},
		Right: &internal.Object{Body: "BODY"},
	}

	for _, tc := range []struct {
		title      string
		args       []string
		color      bool
		pair       *internal.ObjectPair
		noDiff     bool
		exitCode   int
		passedArgs []string
		err        bool
	}{
		{
			title:    "exit_2",
			pair:     diffPair,
			exitCode: diffTestExitFailure,
			err:      true,
		},
		{
			title:    "exit_1",
			pair:     diffPair,
			exitCode: diffTestExitDiff,
			color:    true,
			passedArgs: []string{
				"--unified=" + diffTestDiffContextString,
				"--color=always",
				"--label", "\x1b[33m" + diffTestLeftFile, diffTestObjectID + "\x1b[0m",
				"--label", "\x1b[33m" + diffTestRightFile, diffTestObjectID + "\x1b[0m",
				diffTestAnyString, diffTestAnyString,
			},
		},
		{
			title: "exit_0 additional args",
			pair:  diffPair,
			args: []string{
				"added",
			},
			exitCode: diffTestExitSuccess,
			passedArgs: []string{
				"--unified=" + diffTestDiffContextString,
				"--color=never",
				"--label", diffTestLeftFile, diffTestObjectID,
				"--label", diffTestRightFile, diffTestObjectID,
				"added",
				diffTestAnyString, diffTestAnyString,
			},
		},
		{
			title:    "exit_0",
			pair:     diffPair,
			exitCode: diffTestExitSuccess,
			passedArgs: []string{
				"--unified=" + diffTestDiffContextString,
				"--color=never",
				"--label", diffTestLeftFile, diffTestObjectID,
				"--label", diffTestRightFile, diffTestObjectID,
				diffTestAnyString, diffTestAnyString,
			},
		},
		{
			title:  "nodiff",
			pair:   noDiffPair,
			noDiff: true,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			verifyProcessObjectDiff(
				t,
				tc.args,
				tc.color,
				tc.pair,
				tc.exitCode,
				tc.noDiff,
				tc.err,
				tc.passedArgs,
			)
		})
	}
}
