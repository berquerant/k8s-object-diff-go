package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestEscapeCommand(t *testing.T) {
	for _, tc := range []struct {
		title string
		input string
		want  string
	}{
		{
			title: "string with space",
			input: "foo bar",
			want:  `'foo bar'`,
		},
		{
			title: "safe string without special characters",
			input: "safe",
			want:  "safe",
		},
		{
			title: "string with quotes and symbols",
			input: `echo "hello; world"`,
			want:  `'echo "hello; world"'`,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			assert.Equal(t, tc.want, internal.EscapeCommand(tc.input))
		})
	}
}
