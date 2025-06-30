package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

func TestObjectPair(t *testing.T) {
	const sep = ">"

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, internal.NewObjectPairMap(
			internal.NewObjectMap(sep),
			internal.NewObjectMap(sep),
		).ObjectPairs())
	})

	t.Run("pairs", func(t *testing.T) {
		var (
			left1 = &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "v1",
					Kind:       "k1",
					Metadata: internal.ObjectMeta{
						Namespace: "default",
						Name:      "n1",
					},
				},
			}
			right1 = &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "v1",
					Kind:       "k1",
					Metadata: internal.ObjectMeta{
						Namespace: "default",
						Name:      "n3",
					},
				},
			}
			common1 = &internal.Object{
				Header: internal.ObjectHeader{
					APIVersion: "v1",
					Kind:       "k1",
					Metadata: internal.ObjectMeta{
						Namespace: "default",
						Name:      "n2",
					},
				},
			}
		)

		const (
			left1id   = "v1>k1>default>n1"
			right1id  = "v1>k1>default>n3"
			common1id = "v1>k1>default>n2"
		)

		left := internal.NewObjectMap(sep)
		right := internal.NewObjectMap(sep)

		left.Add(left1)
		right.Add(right1)
		left.Add(common1)
		right.Add(common1)

		m := internal.NewObjectPairMap(left, right)
		got := m.ObjectPairs()
		if !assert.Len(t, got, 3) {
			return
		}
		for _, x := range got {
			if !assert.False(t, x.IsMissing()) {
				return
			}
		}

		p1 := got[0]
		assert.Equal(t, left1id, p1.ID)
		assert.Equal(t, left1id, p1.Left.Header.IntoID(sep))
		assert.Nil(t, p1.Right)

		p2 := got[1]
		assert.Equal(t, common1id, p2.ID)
		assert.Equal(t, common1id, p2.Left.Header.IntoID(sep))
		assert.Equal(t, common1id, p2.Right.Header.IntoID(sep))

		p3 := got[2]
		assert.Equal(t, right1id, p3.ID)
		assert.Equal(t, right1id, p3.Right.Header.IntoID(sep))
		assert.Nil(t, p3.Left)
	})

}
