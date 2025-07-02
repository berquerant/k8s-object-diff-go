package internal_test

import (
	"context"
	"strings"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

type mockLoadObjectFromMapkMarshaler struct{}

func (mockLoadObjectFromMapkMarshaler) Marshal(_ context.Context, _ any) ([]byte, error) {
	return []byte("mocked"), nil
}

func TestLoadObjects(t *testing.T) {
	t.Run("pod", func(t *testing.T) {
		const manifest = `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
`
		r := strings.NewReader(manifest)
		got, err := internal.LoadObjects(context.TODO(), r, internal.NewYamlMarshaler(2, true), true)
		if !assert.Nil(t, err) {
			return
		}
		if !assert.Len(t, got, 1) {
			return
		}
		assert.Equal(t, &internal.Object{
			Header: internal.ObjectHeader{
				APIVersion: "v1",
				Kind:       "Pod",
				Metadata: internal.ObjectMeta{
					Name: "nginx",
				},
			},
			Body: manifest,
		}, got[0])
	})
}

func TestLoadObjectFromMap(t *testing.T) {
	marshaler := &mockLoadObjectFromMapkMarshaler{}

	for _, tc := range []struct {
		title string
		obj   map[string]any
		err   bool
		want  *internal.Object
	}{
		{
			title: "empty",
			obj:   map[string]any{},
			err:   true,
		},
		{
			title: "deployment",
			obj: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"namespace": "default",
					"name":      "test",
				},
			},
			want: &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: internal.ObjectMeta{
						Namespace: "default",
						Name:      "test",
					},
				},
				Body: "mocked",
			},
		},
		{
			title: "clusterrole",
			obj: map[string]any{
				"apiVersion": "rbac.authorization.k8s.io/v1",
				"kind":       "ClusterRole",
				"metadata": map[string]any{
					"name": "test",
				},
			},
			want: &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "ClusterRole",
					Metadata: internal.ObjectMeta{
						Name: "test",
					},
				},
				Body: "mocked",
			},
		},
		{
			title: "no apiVersion",
			obj: map[string]any{
				"kind": "Deployment",
				"metadata": map[string]any{
					"namespace": "default",
					"name":      "test",
				},
			},
			err: true,
		},
		{
			title: "no kind",
			obj: map[string]any{
				"apiVersion": "apps/v1",
				"metadata": map[string]any{
					"namespace": "default",
					"name":      "test",
				},
			},
			err: true,
		},
		{
			title: "no metadata",
			obj: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
			},
			err: true,
		},
		{
			title: "no name",
			obj: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"namespace": "default",
				},
			},
			err: true,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got, err := internal.LoadObjectFromMap(context.TODO(), marshaler, tc.obj)
			if tc.err {
				assert.NotNil(t, err)
				return
			}
			if !assert.Nil(t, err) {
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
