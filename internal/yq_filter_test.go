package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYqFilter(t *testing.T) {
	t.Run("nil or empty filter returns original body", func(t *testing.T) {
		f := internal.NewYqFilter(nil)
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
  labels:
    app: nginx
    buildId: "123"
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: "{}"
    note: "important"
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
        buildId: "123"
      annotations:
        kubectl.kubernetes.io/last-applied-configuration: "{}"
status:
  availableReplicas: 3
`

	t.Run("FieldFilter", func(t *testing.T) {
		f := internal.NewFieldFilter([]string{"metadata.resourceVersion", "status"})
		require.NotNil(t, f)
		res, err := f.FilterBody(inputYAML)
		require.NoError(t, err)
		assert.NotContains(t, res, "resourceVersion")
		assert.NotContains(t, res, "status")
		assert.Contains(t, res, "nginx")
	})

	t.Run("LabelFilter", func(t *testing.T) {
		f := internal.NewLabelFilter([]string{"buildId"})
		require.NotNil(t, f)
		res, err := f.FilterBody(inputYAML)
		require.NoError(t, err)
		assert.NotContains(t, res, "buildId")
		assert.Contains(t, res, "app: nginx")
	})

	t.Run("AnnotationFilter", func(t *testing.T) {
		f := internal.NewAnnotationFilter([]string{"kubectl.kubernetes.io/last-applied-configuration"})
		require.NotNil(t, f)
		res, err := f.FilterBody(inputYAML)
		require.NoError(t, err)
		assert.NotContains(t, res, "last-applied-configuration")
		assert.Contains(t, res, "note: \"important\"")
	})
}
