package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestTailString(t *testing.T) {
	const sep = "."
	for _, tc := range []struct {
		title string
		s     string
		n     int
		want  string
	}{
		{
			title: "four",
			s:     "1.2.3.",
			n:     4,
			want:  "1.2.3.",
		},
		{
			title: "three",
			s:     "1.2.3.",
			n:     3,
			want:  "1.2.3.",
		},
		{
			title: "two",
			s:     "1.2.3.",
			n:     2,
			want:  "2.3.",
		},
		{
			title: "one",
			s:     "1.2.3.",
			n:     1,
			want:  "3.",
		},
		{
			title: "zero",
			s:     "1.2.3.",
			n:     0,
			want:  "",
		},
		{
			title: "empty",
			s:     "",
			n:     1,
			want:  "",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got := internal.TailString(tc.s, sep, tc.n)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHeadString(t *testing.T) {
	const sep = "."
	for _, tc := range []struct {
		title string
		s     string
		n     int
		want  string
	}{
		{
			title: "four",
			s:     "1.2.3.",
			n:     4,
			want:  "1.2.3.",
		},
		{
			title: "three",
			s:     "1.2.3.",
			n:     3,
			want:  "1.2.3.",
		},
		{
			title: "two",
			s:     "1.2.3.",
			n:     2,
			want:  "1.2.",
		},
		{
			title: "one",
			s:     "1.2.3.",
			n:     1,
			want:  "1.",
		},
		{
			title: "zero",
			s:     "1.2.3.",
			n:     0,
			want:  "",
		},
		{
			title: "empty",
			s:     "",
			n:     1,
			want:  "",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got := internal.HeadString(tc.s, sep, tc.n)
			assert.Equal(t, tc.want, got)
		})
	}
}
