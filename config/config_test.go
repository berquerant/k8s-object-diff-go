package config

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigOutMode(t *testing.T) {
	for _, tc := range []struct {
		title string
		out   string
		want  OutMode
	}{
		{title: "text", out: "text", want: OutModeText},
		{title: "yaml", out: "yaml", want: OutModeYaml},
		{title: "id", out: "id", want: OutModeID},
		{title: "idlist", out: "idlist", want: OutModeIDList},
		{title: "markdown", out: "markdown", want: OutModeMarkdown},
		{title: "template", out: "template", want: OutModeTemplate},
		{title: "unknown", out: "unknown", want: OutModeUnknown},
		{title: "empty", out: "", want: OutModeUnknown},
	} {
		t.Run(tc.title, func(t *testing.T) {
			c := &Config{Out: tc.out}
			assert.Equal(t, tc.want, c.OutMode())
		})
	}
}

func TestConfigDiffCommand(t *testing.T) {
	diffPath, err := exec.LookPath("diff")
	require.NoError(t, err, "system 'diff' binary should exist for testing")

	for _, tc := range []struct {
		title       string
		diffCommand string
		wantCmd     []string
		wantErr     error
		wantErrIs   bool
	}{
		{
			title:       "empty command",
			diffCommand: "",
			wantErr:     ErrNoDiffCommand,
			wantErrIs:   true,
		},
		{
			title:       "whitespace only",
			diffCommand: "   ",
			wantErr:     ErrNoDiffCommand,
			wantErrIs:   true,
		},
		{
			title:       "single command without args",
			diffCommand: "diff",
			wantCmd:     []string{diffPath},
		},
		{
			title:       "command with multiple arguments",
			diffCommand: "diff -u --ignore-all-space",
			wantCmd:     []string{diffPath, "-u", "--ignore-all-space"},
		},
		{
			title:       "command with extra whitespace",
			diffCommand: "  diff   -u   -w  ",
			wantCmd:     []string{diffPath, "-u", "-w"},
		},
		{
			title:       "nonexistent command",
			diffCommand: "nonexistent-diff-binary-xyz123",
			wantErr:     exec.ErrNotFound,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			c := &Config{DiffCommand: tc.diffCommand}
			got, err := c.diffCommand()
			if tc.wantErr != nil {
				if tc.wantErrIs {
					assert.ErrorIs(t, err, tc.wantErr)
				} else {
					assert.Error(t, err)
				}
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantCmd, got)
		})
	}
}

func TestConfigNewDiffer(t *testing.T) {
	for _, tc := range []struct {
		title       string
		diffCommand string
		wantType    string
		wantErr     bool
	}{
		{
			title:       "default differs with DMPDiffer",
			diffCommand: "",
			wantType:    "*internal.DMPDiffer",
		},
		{
			title:       "custom diffCommand uses ProcessDiffer",
			diffCommand: "diff -u",
			wantType:    "*internal.ProcessDiffer",
		},
		{
			title:       "invalid command returns error",
			diffCommand: "nonexistent-diff-binary-xyz123",
			wantErr:     true,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			c := &Config{DiffCommand: tc.diffCommand}
			d, err := c.newDiffer()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, d)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, d)
			assert.Equal(t, tc.wantType, fmt.Sprintf("%T", d))
		})
	}
}

func TestConfigClone(t *testing.T) {
	for _, tc := range []struct {
		title string
		cfg   *Config
	}{
		{
			title: "nil config",
			cfg:   nil,
		},
		{
			title: "empty config",
			cfg:   &Config{},
		},
		{
			title: "populated config with slices",
			cfg: &Config{
				Context:             5,
				Separator:           "/",
				Indent:              4,
				Out:                 "yaml",
				Color:               true,
				Labels:              []string{"lbl1", "lbl2"},
				IgnoreMatchingLines: []string{"^#", "test"},
				IgnoreFields:        []string{"spec.replicas"},
				IgnoreLabels:        []string{"app"},
				IgnoreAnnotations:   []string{"build"},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got := tc.cfg.Clone()
			if tc.cfg == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.cfg, got)

			// Verify mutating clone slice contents does not affect original
			if len(tc.cfg.Labels) > 0 {
				got.Labels[0] = "mutated"
				assert.NotEqual(t, tc.cfg.Labels[0], got.Labels[0])
			}
			if len(tc.cfg.IgnoreMatchingLines) > 0 {
				got.IgnoreMatchingLines = append(got.IgnoreMatchingLines, "extra")
				assert.NotContains(t, tc.cfg.IgnoreMatchingLines, "extra")
			}
			if len(tc.cfg.IgnoreFields) > 0 {
				got.IgnoreFields = append(got.IgnoreFields, "extra")
				assert.NotContains(t, tc.cfg.IgnoreFields, "extra")
			}
			if len(tc.cfg.IgnoreLabels) > 0 {
				got.IgnoreLabels = append(got.IgnoreLabels, "extra")
				assert.NotContains(t, tc.cfg.IgnoreLabels, "extra")
			}
			if len(tc.cfg.IgnoreAnnotations) > 0 {
				got.IgnoreAnnotations = append(got.IgnoreAnnotations, "extra")
				assert.NotContains(t, tc.cfg.IgnoreAnnotations, "extra")
			}
		})
	}
}

func TestMarkdownCodeFence(t *testing.T) {
	for _, tc := range []struct {
		title   string
		content string
		want    string
	}{
		{
			title:   "no backticks",
			content: "hello world\n+ foo: bar\n",
			want:    "```",
		},
		{
			title:   "single backtick",
			content: "`inline code`",
			want:    "```",
		},
		{
			title:   "double backticks",
			content: "``code``",
			want:    "```",
		},
		{
			title:   "triple backticks",
			content: "```bash\necho 1\n```",
			want:    "````",
		},
		{
			title:   "quad backticks",
			content: "````markdown\n```\n````",
			want:    "`````",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			assert.Equal(t, tc.want, markdownCodeFence(tc.content))
		})
	}
}

func TestDiffPrinterModes(t *testing.T) {
	pairs := []*internal.ObjectPair{
		{
			ID: "v1>Pod>default>nginx",
			Left: &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "v1",
					Kind:       "Pod",
					Metadata: internal.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
					},
				},
				Body: "apiVersion: v1\nkind: Pod\n",
			},
			Right: &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "v1",
					Kind:       "Pod",
					Metadata: internal.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
					},
				},
				Body: "apiVersion: v1\nkind: Pod\nmetadata:\n  name: nginx\n",
			},
		},
	}

	for _, tc := range []struct {
		title        string
		mode         OutMode
		wantContains []string
		wantErr      error
	}{
		{
			title:        "idlist mode",
			mode:         OutModeIDList,
			wantContains: []string{"v1>Pod>default>nginx"},
			wantErr:      nil,
		},
		{
			title:        "id mode",
			mode:         OutModeID,
			wantContains: []string{},
			wantErr:      nil,
		},
		{
			title:        "yaml mode",
			mode:         OutModeYaml,
			wantContains: []string{"id: v1>Pod>default>nginx", "type: change"},
			wantErr:      ErrDiffFound,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			var buf bytes.Buffer
			differ := internal.NewDMPDiffer()
			p := &diffPrinter{
				mode:         tc.mode,
				pairs:        pairs,
				differ:       differ,
				objectDiffer: internal.NewObjectDiffBuilder(differ, "left", "right", 3, false),
				marshaler:    internal.NewYamlMarshaler(2, false),
				left:         "left",
				right:        "right",
				out:          &buf,
			}
			err := p.print(context.Background())
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			for _, s := range tc.wantContains {
				assert.Contains(t, buf.String(), s)
			}
		})
	}
}

