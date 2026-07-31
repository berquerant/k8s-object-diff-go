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
