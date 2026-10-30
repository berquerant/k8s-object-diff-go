package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestEscapeCommand(t *testing.T) {
	assert.Equal(t, `'foo bar'`, internal.EscapeCommand("foo bar"))
	assert.Equal(t, `safe`, internal.EscapeCommand("safe"))
}
