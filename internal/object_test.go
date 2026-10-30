package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestObjectHeader(t *testing.T) {
	for _, tc := range []struct {
		title string
		obj   internal.ObjectHeader
		sep   string
		want  string
	}{
		{
			title: "normal",
			obj: internal.ObjectHeader{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Metadata: internal.ObjectMeta{
					Namespace: "default",
					Name:      "test",
				},
			},
			sep:  ">",
			want: "apps/v1>Deployment>default>test",
		},
		{
			title: "no namespace",
			obj: internal.ObjectHeader{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
				Metadata: internal.ObjectMeta{
					Name: "test",
				},
			},
			sep:  ">",
			want: "rbac.authorization.k8s.io/v1>ClusterRole>>test",
		},
		{
			title: "escape separator in name",
			obj: internal.ObjectHeader{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Metadata: internal.ObjectMeta{
					Namespace: "ns",
					Name:      "foo>bar",
				},
			},
			sep:  ">",
			want: `v1>ConfigMap>ns>foo\>bar`,
		},
		{
			title: "custom separator hyphen escaping",
			obj: internal.ObjectHeader{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Metadata: internal.ObjectMeta{
					Namespace: "my-ns",
					Name:      "app-name",
				},
			},
			sep:  "-",
			want: `apps/v1-Deployment-my\-ns-app\-name`,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got := tc.obj.IntoID(tc.sep)
			assert.Equal(t, tc.want, got)
		})
	}
}
