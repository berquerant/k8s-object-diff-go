package internal_test

import (
	"sort"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestObjectMap(t *testing.T) {
	m := internal.NewObjectMap(">")
	assert.False(t, m.Add(&internal.Object{
		Header: internal.ObjectHeader{
			APIVersion: "v1",
			Kind:       "k1",
			Metadata: internal.ObjectMeta{
				Namespace: "default",
				Name:      "n1",
			},
		},
	}))
	{
		_, ok := m.Get("v1>k1>default>n1")
		assert.True(t, ok)
	}
	assert.Equal(t, []string{"v1>k1>default>n1"}, m.Keys())

	assert.False(t, m.Add(&internal.Object{
		Header: internal.ObjectHeader{
			APIVersion: "v1",
			Kind:       "k1",
			Metadata: internal.ObjectMeta{
				Namespace: "default",
				Name:      "n3",
			},
		},
	}))
	assert.True(t, m.Add(&internal.Object{
		Header: internal.ObjectHeader{
			APIVersion: "v1",
			Kind:       "k1",
			Metadata: internal.ObjectMeta{
				Namespace: "default",
				Name:      "n1",
			},
		},
	}))
	assert.False(t, m.Add(&internal.Object{
		Header: internal.ObjectHeader{
			APIVersion: "v1",
			Kind:       "k1",
			Metadata: internal.ObjectMeta{
				Namespace: "default",
				Name:      "n2",
			},
		},
	}))

	{
		keys := m.Keys()
		sort.Strings(keys)
		assert.Equal(t, []string{
			"v1>k1>default>n1",
			"v1>k1>default>n2",
			"v1>k1>default>n3",
		}, keys)
	}
}
