package internal_test

import (
	"sort"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestObjectMap(t *testing.T) {
	newObj := func(apiVersion, kind, namespace, name string) *internal.Object {
		return &internal.Object{
			Header: internal.ObjectHeader{
				APIVersion: apiVersion,
				Kind:       kind,
				Metadata: internal.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
			},
		}
	}

	t.Run("Add", func(t *testing.T) {
		m := internal.NewObjectMap(">")
		for _, tc := range []struct {
			title     string
			obj       *internal.Object
			wantExist bool
		}{
			{
				title:     "add first object",
				obj:       newObj("v1", "Pod", "default", "p1"),
				wantExist: false,
			},
			{
				title:     "add second distinct object",
				obj:       newObj("v1", "Pod", "default", "p2"),
				wantExist: false,
			},
			{
				title:     "add duplicate object overwrites and returns true",
				obj:       newObj("v1", "Pod", "default", "p1"),
				wantExist: true,
			},
			{
				title:     "add third object",
				obj:       newObj("v1", "Service", "default", "s1"),
				wantExist: false,
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				got := m.Add(tc.obj)
				assert.Equal(t, tc.wantExist, got)
			})
		}
	})

	t.Run("Get and Keys", func(t *testing.T) {
		m := internal.NewObjectMap(">")
		m.Add(newObj("v1", "Pod", "default", "p1"))
		m.Add(newObj("v1", "Pod", "default", "p2"))

		for _, tc := range []struct {
			title   string
			id      string
			wantOk  bool
			wantObj bool
		}{
			{
				title:   "get existing object p1",
				id:      "v1>Pod>default>p1",
				wantOk:  true,
				wantObj: true,
			},
			{
				title:   "get existing object p2",
				id:      "v1>Pod>default>p2",
				wantOk:  true,
				wantObj: true,
			},
			{
				title:   "get nonexistent object",
				id:      "v1>Pod>default>p3",
				wantOk:  false,
				wantObj: false,
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				obj, ok := m.Get(tc.id)
				assert.Equal(t, tc.wantOk, ok)
				if tc.wantObj {
					assert.NotNil(t, obj)
				} else {
					assert.Nil(t, obj)
				}
			})
		}

		keys := m.Keys()
		sort.Strings(keys)
		assert.Equal(t, []string{"v1>Pod>default>p1", "v1>Pod>default>p2"}, keys)
	})
}
