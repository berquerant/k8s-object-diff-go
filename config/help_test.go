package config_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/config"
	"github.com/stretchr/testify/assert"
)

var update = flag.Bool("update", false, "update golden files")

func TestHelpMarkdownGolden(t *testing.T) {
	goldenFile := filepath.Join("testdata", "help.md.golden")
	got := config.HelpMarkdown()

	if *update {
		if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenFile)
	if !assert.Nil(t, err) {
		return
	}
	assert.Equal(t, string(want), got)
}
