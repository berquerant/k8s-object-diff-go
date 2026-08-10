package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldFilter(t *testing.T) {
	t.Run("nil or empty paths returns original body", func(t *testing.T) {
		f := internal.NewFieldFilter(nil)
		got, err := f.FilterBody("apiVersion: v1\n")
		require.NoError(t, err)
		assert.Equal(t, "apiVersion: v1\n", got)
	})

	const inputYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  resourceVersion: "12345"
  generation: 2
spec:
  replicas: 3
status:
  availableReplicas: 3
`

	for _, tc := range []struct {
		title string
		paths []string
		check func(t *testing.T, res string)
	}{
		{
			title: "remove single top-level field",
			paths: []string{"status"},
			check: func(t *testing.T, res string) {
				assert.NotContains(t, res, "status")
				assert.Contains(t, res, "resourceVersion")
				assert.Contains(t, res, "nginx")
			},
		},
		{
			title: "remove nested field",
			paths: []string{"metadata.resourceVersion"},
			check: func(t *testing.T, res string) {
				assert.NotContains(t, res, "resourceVersion")
				assert.Contains(t, res, "generation")
				assert.Contains(t, res, "status")
			},
		},
		{
			title: "remove multiple fields",
			paths: []string{"metadata.resourceVersion", "metadata.generation", "status"},
			check: func(t *testing.T, res string) {
				assert.NotContains(t, res, "resourceVersion")
				assert.NotContains(t, res, "generation")
				assert.NotContains(t, res, "status")
				assert.Contains(t, res, "nginx")
				assert.Contains(t, res, "replicas")
			},
		},
		{
			title: "path starting with dot",
			paths: []string{".metadata.resourceVersion"},
			check: func(t *testing.T, res string) {
				assert.NotContains(t, res, "resourceVersion")
				assert.Contains(t, res, "nginx")
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f := internal.NewFieldFilter(tc.paths)
			require.NotNil(t, f)
			res, err := f.FilterBody(inputYAML)
			require.NoError(t, err)
			tc.check(t, res)
		})
	}
}
