package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLineFilter(t *testing.T) {
	for _, tc := range []struct {
		title    string
		patterns []string
		wantErr  bool
	}{
		{
			title:    "valid patterns",
			patterns: []string{"^foo", "bar$"},
			wantErr:  false,
		},
		{
			title:    "nil patterns",
			patterns: nil,
			wantErr:  false,
		},
		{
			title:    "empty patterns",
			patterns: []string{},
			wantErr:  false,
		},
		{
			title:    "invalid regex pattern",
			patterns: []string{"["},
			wantErr:  true,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f, err := internal.NewLineFilter(tc.patterns)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, f)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			}
		})
	}
}

func TestLineFilter(t *testing.T) {
	for _, tc := range []struct {
		title    string
		patterns []string
		input    string
		want     string
	}{
		{
			title:    "no patterns returns unchanged",
			patterns: nil,
			input:    "a\nb\n",
			want:     "a\nb\n",
		},
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
