package config_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/config"
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
