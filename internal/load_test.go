package internal_test

import (
	"context"
	"strings"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLoadObjectFromMapkMarshaler struct{}

func (mockLoadObjectFromMapkMarshaler) Marshal(_ context.Context, _ any) ([]byte, error) {
	return []byte("mocked"), nil
}

func TestLoadObjects(t *testing.T) {
	const podManifest = `apiVersion: v1
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
	const multiDocManifest = `apiVersion: v1
kind: Service
metadata:
  name: svc1
---
apiVersion: v1
kind: Service
metadata:
  name: svc2
`

	for _, tc := range []struct {
		title    string
		manifest string
		wantLen  int
		wantErr  bool
	}{
		{
			title:    "single pod manifest",
			manifest: podManifest,
			wantLen:  1,
		},
		{
			title:    "multi-document manifest",
			manifest: multiDocManifest,
			wantLen:  2,
		},
		{
			title:    "empty manifest",
			manifest: "",
			wantLen:  0,
		},
		{
			title:    "invalid manifest missing apiVersion",
			manifest: "kind: Pod\nmetadata:\n  name: nginx\n",
			wantErr:  true,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			r := strings.NewReader(tc.manifest)
			got, err := internal.LoadObjects(context.TODO(), r, internal.NewYamlMarshaler(2, true), true)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tc.wantLen)
			if tc.wantLen > 0 {
				assert.NotEmpty(t, got[0].Header.Kind)
			}
		})
	}
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
