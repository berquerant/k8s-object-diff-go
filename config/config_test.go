package config

import (
	"fmt"
	"os/exec"
	"testing"

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
