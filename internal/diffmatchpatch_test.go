package internal_test

import (
	"encoding/json"
	"log/slog"
	"os"
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

func TestDMP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	const (
		leftLabel  = "LEFT"
		rightLabel = "RIGHT"
	)
	newResult := func(patches ...*internal.DMPPatch) *internal.DMPResult {
		return &internal.DMPResult{
			LeftLabel:  leftLabel,
			RightLabel: rightLabel,
			Patches:    patches,
		}
	}
	lines := func(i, j int) string {
		v := make([]string, j-i+1)
		for t := 0; t < len(v); t++ {
			v[t] = strconv.Itoa(t + i)
		}
		return strings.Join(v, "\n") + "\n"
	}
	for _, tc := range []struct {
		title       string
		left        string
		right       string
		contextSize int
		want        *internal.DMPResult
		noDiff      bool
	}{
		{
			title:       "split large",
			left:        lines(1, 3) + lines(4, 4) + lines(6, 12) + lines(13, 14) + lines(21, 30),
			right:       lines(1, 3) + lines(5, 5) + lines(6, 12) + lines(15, 20) + lines(21, 30) + lines(31, 33),
			contextSize: 3,
			want: newResult(&internal.DMPPatch{
				LeftStart:   1,
				LeftLength:  7,
				RightStart:  1,
				RightLength: 7,
				Hunks: []*internal.DMPHunk{
					{
						Op:   internal.DMPOpEqual,
						Body: lines(1, 3),
					},
					{
						Op:   internal.DMPOpDelete,
						Body: lines(4, 4),
					},
					{
						Op:   internal.DMPOpInsert,
						Body: lines(5, 5),
					},
					{
						Op:   internal.DMPOpEqual,
						Body: lines(6, 8),
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
						Body: lines(10, 12),
					},
					{
						Op:   internal.DMPOpDelete,
						Body: lines(13, 14),
					},
					{
						Op:   internal.DMPOpInsert,
						Body: lines(15, 20),
					},
					{
						Op:   internal.DMPOpEqual,
						Body: lines(21, 23),
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
						Body: lines(28, 30),
					},
					{
						Op:   internal.DMPOpInsert,
						Body: lines(31, 33),
					},
				},
			}),
		},
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
	} {
		t.Run(tc.title, func(t *testing.T) {
			dmp := &internal.DMP{
				LeftLabel:  leftLabel,
				RightLabel: rightLabel,
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
