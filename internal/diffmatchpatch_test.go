package internal_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestDMPHunk(t *testing.T) {
	t.Run("IntoString", func(t *testing.T) {
		for _, tc := range []struct {
			title string
			hunk  *internal.DMPHunk
			want  string
		}{
			{
				title: "equal",
				hunk: &internal.DMPHunk{
					Op: internal.DMPOpEqual,
					Body: `line1
line2
`,
				},
				want: ` line1
 line2
`,
			},
			{
				title: "insert",
				hunk: &internal.DMPHunk{
					Op: internal.DMPOpInsert,
					Body: `line1
line2
`,
				},
				want: `+line1
+line2
`,
			},
			{
				title: "delete",
				hunk: &internal.DMPHunk{
					Op: internal.DMPOpDelete,
					Body: `line1
line2
`,
				},
				want: `-line1
-line2
`,
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				got := tc.hunk.IntoString(false)
				assert.Equal(t, tc.want, got)
			})
		}
	})
}

const (
	dmpTestLeftLabel  = "LEFT"
	dmpTestRightLabel = "RIGHT"
)

type dmpTestCase struct {
	title       string
	left        string
	right       string
	contextSize int
	want        *internal.DMPResult
	noDiff      bool
}

func newDMPResult(patches ...*internal.DMPPatch) *internal.DMPResult {
	return &internal.DMPResult{
		LeftLabel:  dmpTestLeftLabel,
		RightLabel: dmpTestRightLabel,
		Patches:    patches,
	}
}

func dmpLines(i, j int) string {
	v := make([]string, j-i+1)
	for t := range v {
		v[t] = strconv.Itoa(t + i)
	}
	return strings.Join(v, "\n") + "\n"
}

func runDMPTests(t *testing.T, cases []dmpTestCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			dmp := &internal.DMP{
				LeftLabel:  dmpTestLeftLabel,
				RightLabel: dmpTestRightLabel,
				Context:    tc.contextSize,
			}
			got, err := dmp.Diff(tc.left, tc.right)
			if tc.noDiff {
				assert.ErrorIs(t, err, internal.ErrDMPNoDiff, tc.title)
				return
			}
			if !assert.Nil(t, err, tc.title) {
				return
			}

			wantJSON, _ := json.Marshal(tc.want)
			gotJSON, _ := json.Marshal(got)
			assert.Equal(t, tc.want, got, "%s:\nWANT\n%s\nGOT\n%s", tc.title, wantJSON, gotJSON)
		})
	}
}

func TestDMP_SplitLarge(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title:       "split large",
			left:        dmpLines(1, 3) + dmpLines(4, 4) + dmpLines(6, 12) + dmpLines(13, 14) + dmpLines(21, 30),
			right:       dmpLines(1, 3) + dmpLines(5, 5) + dmpLines(6, 12) + dmpLines(15, 20) + dmpLines(21, 30) + dmpLines(31, 33),
			contextSize: 3,
			want: newDMPResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  7,
				RightStart:  1,
				RightLength: 7,
				Hunks: []*internal.DMPHunk{
					{
						Op:   internal.DMPOpEqual,
						Body: dmpLines(1, 3),
					},
					{
						Op:   internal.DMPOpDelete,
						Body: dmpLines(4, 4),
					},
					{
						Op:   internal.DMPOpInsert,
						Body: dmpLines(5, 5),
					},
					{
						Op:   internal.DMPOpEqual,
						Body: dmpLines(6, 8),
					},
				},
			}, &internal.DMPPatch{
				LeftStart:   9,
				LeftLength:  8,
				RightStart:  9,
				RightLength: 12,
				Hunks: []*internal.DMPHunk{
					{
						Op:   internal.DMPOpEqual,
						Body: dmpLines(10, 12),
					},
					{
						Op:   internal.DMPOpDelete,
						Body: dmpLines(13, 14),
					},
					{
						Op:   internal.DMPOpInsert,
						Body: dmpLines(15, 20),
					},
					{
						Op:   internal.DMPOpEqual,
						Body: dmpLines(21, 23),
					},
				},
			}, &internal.DMPPatch{
				LeftStart:   21,
				LeftLength:  3,
				RightStart:  25,
				RightLength: 6,
				Hunks: []*internal.DMPHunk{
					{
						Op:   internal.DMPOpEqual,
						Body: dmpLines(28, 30),
					},
					{
						Op:   internal.DMPOpInsert,
						Body: dmpLines(31, 33),
					},
				},
			}),
		},
	})
}

func TestDMP_SplitHunks(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title: "split hunks2",
			left: `line1
line2
line4
line5
line6
line7
line8
`,
			right: `line1
line2
line3
line4
line5
line6
line7
line8
line9
line10
`,
			contextSize: 2,
			want: newDMPResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  4,
				RightStart:  1,
				RightLength: 5,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line1
line2
`,
					},
					{
						Op: internal.DMPOpInsert,
						Body: `line3
`,
					},
					{
						Op: internal.DMPOpEqual,
						Body: `line4
line5
`,
					},
				},
			}, &internal.DMPPatch{
				LeftStart:   6,
				LeftLength:  2,
				RightStart:  7,
				RightLength: 4,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line7
line8
`,
					},
					{
						Op: internal.DMPOpInsert,
						Body: `line9
line10
`,
					},
				},
			}),
		},
	})
}

func newResult(patches ...*internal.DMPPatch) *internal.DMPResult {
	return newDMPResult(patches...)
}

func TestDMP_ContextGrouping(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title: "do not split hunks",
			left: `line1
line2
line4
line5
line6
line7
line8
`,
			right: `line1
line2
line3
line4
line5
line6
line7
`,
			contextSize: 3,
			want: newResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  7,
				RightStart:  1,
				RightLength: 7,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line1
line2
`,
					},
					{
						Op: internal.DMPOpInsert,
						Body: `line3
`,
					},
					{
						Op: internal.DMPOpEqual,
						Body: `line4
line5
line6
line7
`,
					},
					{
						Op: internal.DMPOpDelete,
						Body: `line8
`,
					},
				},
			}),
		},
	})
}

func TestDMP_SplitHunksContext(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title: "split hunks",
			left: `line1
line2
line4
line5
line6
line7
line8
`,
			right: `line1
line2
line3
line4
line5
line6
line7
`,
			contextSize: 1,
			want: newResult(&internal.DMPPatch{
				LeftStart:   2,
				LeftLength:  2,
				RightStart:  2,
				RightLength: 3,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line2
`,
					},
					{
						Op: internal.DMPOpInsert,
						Body: `line3
`,
					},
					{
						Op: internal.DMPOpEqual,
						Body: `line4
`,
					},
				},
			}, &internal.DMPPatch{
				LeftStart:   6,
				LeftLength:  2,
				RightStart:  7,
				RightLength: 1,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line7
`,
					},
					{
						Op: internal.DMPOpDelete,
						Body: `line8
`,
					},
				},
			}),
		},
	})
}

func TestDMP_MiddleInsert(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title: "middle insert",
			left: `line1
line2
line4
line5
`,
			right: `line1
line2
line3
line4
line5
`,
			contextSize: 3,
			want: newResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  4,
				RightStart:  1,
				RightLength: 5,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line1
line2
`,
					},
					{
						Op: internal.DMPOpInsert,
						Body: `line3
`,
					},
					{
						Op: internal.DMPOpEqual,
						Body: `line4
line5
`,
					},
				},
			}),
		},
	})
}

func TestDMP_EdgeCases(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title: "ends with delete",
			left: `line1
line2
line3
line4
`,
			right: `line1
line2
line3
`,
			contextSize: 3,
			want: newResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  4,
				RightStart:  1,
				RightLength: 3,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpEqual,
						Body: `line1
line2
line3
`,
					},
					{
						Op: internal.DMPOpDelete,
						Body: `line4
`,
					},
				},
			}),
		},
		{
			title: "starts with insert",
			left: `line1
line2
line3
`,
			right: `line0
line1
line2
line3
`,
			contextSize: 3,
			want: newResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  3,
				RightStart:  1,
				RightLength: 4,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpInsert,
						Body: `line0
`,
					},
					{
						Op: internal.DMPOpEqual,
						Body: `line1
line2
line3
`,
					},
				},
			}),
		},
	})
}

func TestDMP_Boundaries(t *testing.T) {
	runDMPTests(t, []dmpTestCase{
		{
			title:       "all empty",
			contextSize: 3,
			noDiff:      true,
		},
		{
			title: "all equal",
			left: `line1
line2
line3
`,
			right: `line1
line2
line3
`,
			contextSize: 3,
			noDiff:      true,
		},
		{
			title: "all insert",
			left:  ``,
			right: `line1
line2
line3
`,
			contextSize: 3,
			want: newResult(&internal.DMPPatch{
				LeftStart:   0,
				LeftLength:  0,
				RightStart:  1,
				RightLength: 3,
				Hunks: []*internal.DMPHunk{
					{
						Op: internal.DMPOpInsert,
						Body: `line1
line2
line3
`,
					},
				},
			}),
		},
	})
}

func TestDMPDiffContext(t *testing.T) {
	for _, tc := range []struct {
		title     string
		left      string
		right     string
		cancel    bool
		wantErrIs error
		checkDiff bool
	}{
		{
			title:     "canceled context returns context.Canceled",
			left:      "left text\n",
			right:     "right text\n",
			cancel:    true,
			wantErrIs: context.Canceled,
		},
		{
			title:     "active context with differences succeeds",
			left:      "left text\n",
			right:     "right text\n",
			cancel:    false,
			checkDiff: true,
		},
		{
			title:     "active context with identical inputs returns ErrDMPNoDiff",
			left:      "same text\n",
			right:     "same text\n",
			cancel:    false,
			wantErrIs: internal.ErrDMPNoDiff,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()
			if tc.cancel {
				cancelFunc()
			}
			dmp := &internal.DMP{
				LeftLabel:  "left",
				RightLabel: "right",
				Context:    3,
			}
			res, err := dmp.DiffContext(ctx, tc.left, tc.right)
			if tc.wantErrIs != nil {
				assert.ErrorIs(t, err, tc.wantErrIs)
				return
			}
			assert.NoError(t, err)
			if tc.checkDiff {
				assert.NotNil(t, res)
				assert.NotEmpty(t, res.Patches)
			}
		})
	}
}
