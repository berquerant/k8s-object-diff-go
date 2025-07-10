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

func TestProcessObjectDiffBuilder(t *testing.T) {
	const (
		command           = "./diff_test.sh"
		anyString         = "ANY"
		leftFile          = "LEFT"
		rightFile         = "RIGHT"
		diffContext       = 3
		diffContextString = "3"
		objectID          = "OBJECT_ID"

		diffTestExitCode = "EXIT_CODE"
		exitSuccess      = 0
		exitDiff         = 1
		exitFailure      = 2
	)
	var (
		noDiffPair = &internal.ObjectPair{
			ID:    objectID,
			Left:  &internal.Object{},
			Right: &internal.Object{},
		}
		diffPair = &internal.ObjectPair{
			ID:   objectID,
			Left: &internal.Object{},
			Right: &internal.Object{
				Body: "BODY",
			},
		}
	)

	for _, tc := range []struct {
		title string
		args  []string
		// left        string
		// right       string
		// diffContext int
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
			exitCode: exitFailure,
			err:      true,
		},
		{
			title:    "exit_1",
			pair:     diffPair,
			exitCode: exitDiff,
			color:    true,
			passedArgs: []string{
				"--unified=" + diffContextString,
				"--color=always",
				"--label", "\x1b[33m" + leftFile, objectID + "\x1b[0m", // due to strings.Split
				"--label", "\x1b[33m" + rightFile, objectID + "\x1b[0m", // due to strings.Split
				anyString, anyString,
			},
		},
		{
			title: "exit_0 additional args",
			pair:  diffPair,
			args: []string{
				"added",
			},
			exitCode: exitSuccess,
			passedArgs: []string{
				"--unified=" + diffContextString,
				"--color=never",
				"--label", leftFile, objectID, // due to strings.Split
				"--label", rightFile, objectID, // due to strings.Split
				"added",
				anyString, anyString,
			},
		},
		{
			title:    "exit_0",
			pair:     diffPair,
			exitCode: exitSuccess,
			passedArgs: []string{
				"--unified=" + diffContextString,
				"--color=never",
				"--label", leftFile, objectID, // due to strings.Split
				"--label", rightFile, objectID, // due to strings.Split
				anyString, anyString,
			},
		},
		{
			title:  "nodiff",
			pair:   noDiffPair,
			noDiff: true,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			_ = os.Setenv(diffTestExitCode, strconv.Itoa(tc.exitCode))
			defer os.Unsetenv(diffTestExitCode)

			x := internal.NewObjectDiffBuilder(
				internal.NewProcessDiffer(command, tc.args),
				leftFile, rightFile,
				diffContext,
				tc.color,
			)
			got, err := x.ObjectDiff(context.TODO(), tc.pair)
			if tc.err {
				assert.NotNil(t, err)
				return
			}
			if !assert.Nil(t, err) {
				return
			}
			assert.Equal(t, tc.pair, got.Pair)
			if tc.noDiff {
				assert.Empty(t, got.Diff)
				return
			}
			passed := strings.Split(got.Diff, " ")
			if !assert.Equal(t, len(tc.passedArgs), len(passed),
				"want:%#v\ngot:%#v", tc.passedArgs, passed,
			) {
				return
			}
			for i, w := range tc.passedArgs {
				g := passed[i]
				if w == anyString {
					continue
				}
				assert.Equal(t, w, g, "passed[%i], i")
			}
		})
	}
}
