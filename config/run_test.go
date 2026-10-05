package config_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	chartRepoName       = "external-secrets"
	chartRepoURL        = "https://charts.external-secrets.io"
	chartTemplateName   = "external-secrets"
	chartName           = "external-secrets/external-secrets"
	leftChartVersion    = "0.10.7"
	rightChartVersion   = "0.18.2"
	externalDiffCommand = "diff"
	externalHelmCommand = "helm"
)

func renderChart(filename, version string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	cmdPath, err := exec.LookPath(externalHelmCommand)
	if err != nil {
		return err
	}

	if err := exec.Command(cmdPath, "repo", "add", chartRepoName, chartRepoURL).Run(); err != nil {
		return err
	}

	cmd := exec.Command(cmdPath, "template", chartTemplateName, chartName, "--version", version)
	cmd.Stdout = f
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func BenchmarkRun(b *testing.B) {
	dir := b.TempDir()
	left := filepath.Join(dir, "eso-"+leftChartVersion+".yml")
	right := filepath.Join(dir, "eso-"+rightChartVersion+".yml")

	if _, err := exec.LookPath(externalDiffCommand); err != nil {
		b.Fatal(err)
	}
	if err := renderChart(left, leftChartVersion); err != nil {
		b.Fatal(err)
	}
	if err := renderChart(right, rightChartVersion); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.Run("builtin", func(b *testing.B) {
		b.ReportAllocs()
		c := &config.Config{
			Quiet:             true,
			AllowDuplicateKey: true,
			Out:               string(config.OutModeText),
		}
		for b.Loop() {
			_ = c.Run(io.Discard, left, right)
		}
	})
	b.Run("external", func(b *testing.B) {
		b.ReportAllocs()
		c := &config.Config{
			Quiet:             true,
			AllowDuplicateKey: true,
			Out:               string(config.OutModeText),
			DiffCommand:       externalDiffCommand,
		}
		for b.Loop() {
			_ = c.Run(io.Discard, left, right)
		}
	})
}

func TestStdinInput(t *testing.T) {
	t.Run("2 stdins", func(t *testing.T) {
		var c config.Config
		assert.ErrorContains(t, c.Run(io.Discard, "-", "-"), "cannot be specified for both left and right")
	})

	const (
		manifest1 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  os: debian`
		manifest2 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  os: ubuntu`
	)
	file := filepath.Join(t.TempDir(), "test.yml")
	if !assert.Nil(t, os.WriteFile(file, []byte(manifest1), 0644)) {
		return
	}
	newStdin := func() io.Reader {
		return bytes.NewBufferString(manifest2)
	}

	for _, tc := range []struct {
		name        string
		left, right string
		want        string
	}{
		{
			name:  "right stdin",
			left:  file,
			right: "-",
			want: fmt.Sprintf(`--- %s v1>ConfigMap>>test
+++ - v1>ConfigMap>>test
@@ -1,6 +1,6 @@
 apiVersion: v1
 data:
-  os: debian
+  os: ubuntu
 kind: ConfigMap
 metadata:
   name: test
`, file),
		},
		{
			name:  "left stdin",
			left:  "-",
			right: file,
			want: fmt.Sprintf(`--- - v1>ConfigMap>>test
+++ %s v1>ConfigMap>>test
@@ -1,6 +1,6 @@
 apiVersion: v1
 data:
-  os: ubuntu
+  os: debian
 kind: ConfigMap
 metadata:
   name: test
`, file),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var c config.Config
			c.Separator = ">"
			c.Indent = 2
			c.Context = 3
			c.Stdin = newStdin()
			var got bytes.Buffer
			assert.ErrorIs(t, c.Run(&got, tc.left, tc.right), config.ErrDiffFound)
			assert.Equal(t, tc.want, got.String())
		})
	}
}

func TestMarkdownTitle(t *testing.T) {
	const (
		manifest1 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  os: debian`
		manifest2 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  os: ubuntu`
	)

	dir := t.TempDir()
	leftFile := filepath.Join(dir, "left.yml")
	rightFile := filepath.Join(dir, "right.yml")
	require.NoError(t, os.WriteFile(leftFile, []byte(manifest1), 0o644))
	require.NoError(t, os.WriteFile(rightFile, []byte(manifest2), 0o644))

	t.Run("default title", func(t *testing.T) {
		var c config.Config
		c.Out = string(config.OutModeMarkdown)
		var got bytes.Buffer
		assert.ErrorIs(t, c.Run(&got, leftFile, rightFile), config.ErrDiffFound)
		wantTitle := fmt.Sprintf("# Objdiff Summary: %s <-> %s\n\n`%s` <-> `%s`", leftFile, rightFile, leftFile, rightFile)
		assert.Contains(t, got.String(), wantTitle)
	})

	t.Run("custom title with template variables", func(t *testing.T) {
		var c config.Config
		c.Out = string(config.OutModeMarkdown)
		c.MarkdownTitle = "Comparison of {{ .LEFT_FILE }} and {{ .RIGHT_FILE }}"
		var got bytes.Buffer
		assert.ErrorIs(t, c.Run(&got, leftFile, rightFile), config.ErrDiffFound)
		wantTitle := fmt.Sprintf("# Comparison of %s and %s\n\n`%s` <-> `%s`", leftFile, rightFile, leftFile, rightFile)
		assert.Contains(t, got.String(), wantTitle)
	})

	t.Run("custom title with labels", func(t *testing.T) {
		var c config.Config
		c.Out = string(config.OutModeMarkdown)
		c.MarkdownTitle = "[{{ .LEFT_FILE }}] vs [{{ .RIGHT_FILE }}]"
		c.Labels = []string{"prod", "stg"}
		var got bytes.Buffer
		assert.ErrorIs(t, c.Run(&got, leftFile, rightFile), config.ErrDiffFound)
		wantTitle := "# [prod] vs [stg]\n\n`prod` <-> `stg`"
		assert.Contains(t, got.String(), wantTitle)
	})

	t.Run("invalid title template", func(t *testing.T) {
		var c config.Config
		c.Out = string(config.OutModeMarkdown)
		c.MarkdownTitle = "{{ .INVALID"
		var got bytes.Buffer
		err := c.Run(&got, leftFile, rightFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse markdown title template")
	})
}

func TestMarkdownTemplate(t *testing.T) {
	const (
		manifest1 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  os: debian`
		manifest2 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  os: ubuntu`
	)

	dir := t.TempDir()
	leftFile := filepath.Join(dir, "left.yml")
	rightFile := filepath.Join(dir, "right.yml")
	require.NoError(t, os.WriteFile(leftFile, []byte(manifest1), 0o644))
	require.NoError(t, os.WriteFile(rightFile, []byte(manifest2), 0o644))

	tmplFile := filepath.Join(dir, "template.tmpl")
	require.NoError(t, os.WriteFile(tmplFile, []byte("FILE: {{ .Stats.Change }} changes"), 0o644))

	const defaultReproducedTemplate = `# Objdiff Summary: {{ .Left }} <-> {{ .Right }}

{{ if not .HasDiff -}}
{{ .Left }} <-> {{ .Right }}

No changes.
{{ else -}}
` + "`" + `{{ .Left }}` + "`" + ` <-> ` + "`" + `{{ .Right }}` + "`" + `

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| {{ .Stats.Add }} | {{ .Stats.Change }} | {{ .Stats.Destroy }} |
{{- range .Diffs }}
## {{ .DiffType }} ` + "`" + `{{ .ID }}` + "`" + `

<details><summary>View Diff</summary>

` + "``` diff\n" + `{{ .Diff }}` + "```\n" + `
</details>
{{ end -}}
{{ end }}`

	var defaultDiffBuf bytes.Buffer
	assert.ErrorIs(t, (&config.Config{Out: string(config.OutModeMarkdown)}).Run(&defaultDiffBuf, leftFile, rightFile), config.ErrDiffFound)

	var defaultNoDiffBuf bytes.Buffer
	assert.NoError(t, (&config.Config{Out: string(config.OutModeMarkdown)}).Run(&defaultNoDiffBuf, leftFile, leftFile))

	for _, tc := range []struct {
		name         string
		left, right  string
		template     string
		headingLevel uint
		title        string
		wantErr      error
		errContains  string
		want         string
		wantExact    bool
	}{
		{
			name:      "reproduce default markdown output on diff",
			left:      leftFile,
			right:     rightFile,
			template:  defaultReproducedTemplate,
			wantErr:   config.ErrDiffFound,
			want:      defaultDiffBuf.String(),
			wantExact: true,
		},
		{
			name:      "reproduce default markdown output on nodiff",
			left:      leftFile,
			right:     leftFile,
			template:  defaultReproducedTemplate,
			wantErr:   nil,
			want:      defaultNoDiffBuf.String(),
			wantExact: true,
		},
		{
			name:      "read template from file with @",
			left:      leftFile,
			right:     rightFile,
			template:  "@" + tmplFile,
			wantErr:   config.ErrDiffFound,
			want:      "FILE: 1 changes",
			wantExact: true,
		},
		{
			name:      "escape @ with @@",
			left:      leftFile,
			right:     rightFile,
			template:  "@@at-sign: {{ .Stats.Change }}",
			wantErr:   config.ErrDiffFound,
			want:      "@at-sign: 1",
			wantExact: true,
		},
		{
			name:      "sprig functions available",
			left:      leftFile,
			right:     rightFile,
			template:  `{{ upper "hello" }} | {{ repeat 3 "!" }} | {{ add .Stats.Change 10 }}`,
			wantErr:   config.ErrDiffFound,
			want:      "HELLO | !!! | 11",
			wantExact: true,
		},
		{
			name:         "markdown-heading and markdown-title are ignored",
			left:         leftFile,
			right:        rightFile,
			template:     "CustomOutput: {{ .Stats.Change }}",
			headingLevel: 4,
			title:        "Should be ignored",
			wantErr:      config.ErrDiffFound,
			want:         "CustomOutput: 1",
			wantExact:    true,
		},
		{
			name:        "error reading nonexistent file",
			left:        leftFile,
			right:       rightFile,
			template:    "@" + filepath.Join(dir, "nonexistent.tmpl"),
			errContains: "failed to read markdown template file",
		},
		{
			name:        "error invalid template syntax",
			left:        leftFile,
			right:       rightFile,
			template:    "{{ .Unclosed",
			errContains: "failed to parse markdown template",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var c config.Config
			c.Out = string(config.OutModeMarkdown)
			c.MarkdownTemplate = tc.template
			c.MarkdownHeadingLevel = tc.headingLevel
			c.MarkdownTitle = tc.title

			var got bytes.Buffer
			err := c.Run(&got, tc.left, tc.right)

			if tc.errContains != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}

			if tc.wantExact {
				assert.Equal(t, tc.want, got.String())
			}
		})
	}
}

