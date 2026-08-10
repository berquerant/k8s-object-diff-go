package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineFilter(t *testing.T) {
	t.Run("NewLineFilter invalid pattern", func(t *testing.T) {
		_, err := internal.NewLineFilter([]string{"["})
		assert.Error(t, err)
	})

	t.Run("no patterns returns unchanged", func(t *testing.T) {
		f, err := internal.NewLineFilter(nil)
		require.NoError(t, err)
		assert.Equal(t, "a\nb\n", f.Filter("a\nb\n"))
	})

	for _, tc := range []struct {
		title    string
		patterns []string
		input    string
		want     string
	}{
		{
			title:    "remove matching lines",
			patterns: []string{"foo"},
			input:    "foo\nbar\nfoo: baz\n",
			want:     "bar\n",
		},
		{
			title:    "preserve trailing newline",
			patterns: []string{"^x"},
			input:    "x: 1\ny: 2\n",
			want:     "y: 2\n",
		},
		{
			title:    "no trailing newline",
			patterns: []string{"^x"},
			input:    "x: 1\ny: 2",
			want:     "y: 2",
		},
		{
			title:    "all lines removed",
			patterns: []string{".*"},
			input:    "a\nb\n",
			want:     "",
		},
		{
			title:    "multiple patterns",
			patterns: []string{"^a", "^b"},
			input:    "a: 1\nb: 2\nc: 3\n",
			want:     "c: 3\n",
		},
		{
			title:    "empty input",
			patterns: []string{"x"},
			input:    "",
			want:     "",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f, err := internal.NewLineFilter(tc.patterns)
			require.NoError(t, err)
			assert.Equal(t, tc.want, f.Filter(tc.input))
		})
	}
}
