package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectPair(t *testing.T) {
	const sep = ">"

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

	type expectedPair struct {
		id        string
		hasLeft   bool
		hasRight  bool
		isMissing bool
	}

	for _, tc := range []struct {
		title     string
		leftObjs  []*internal.Object
		rightObjs []*internal.Object
		wantPairs []expectedPair
	}{
		{
			title:     "empty maps produce no pairs",
			leftObjs:  nil,
			rightObjs: nil,
			wantPairs: []expectedPair{},
		},
		{
			title: "left only produces destroy pair",
			leftObjs: []*internal.Object{
				newObj("v1", "Pod", "default", "p1"),
			},
			rightObjs: nil,
			wantPairs: []expectedPair{
				{
					id:        "v1>Pod>default>p1",
					hasLeft:   true,
					hasRight:  false,
					isMissing: false,
				},
			},
		},
		{
			title:    "right only produces add pair",
			leftObjs: nil,
			rightObjs: []*internal.Object{
				newObj("v1", "Pod", "default", "p2"),
			},
			wantPairs: []expectedPair{
				{
					id:        "v1>Pod>default>p2",
					hasLeft:   false,
					hasRight:  true,
					isMissing: false,
				},
			},
		},
		{
			title: "mixed pairs sorted by object ID",
			leftObjs: []*internal.Object{
				newObj("v1", "k1", "default", "n1"),
				newObj("v1", "k1", "default", "n2"),
			},
			rightObjs: []*internal.Object{
				newObj("v1", "k1", "default", "n2"),
				newObj("v1", "k1", "default", "n3"),
			},
			wantPairs: []expectedPair{
				{
					id:        "v1>k1>default>n1",
					hasLeft:   true,
					hasRight:  false,
					isMissing: false,
				},
				{
					id:        "v1>k1>default>n2",
					hasLeft:   true,
					hasRight:  true,
					isMissing: false,
				},
				{
					id:        "v1>k1>default>n3",
					hasLeft:   false,
					hasRight:  true,
					isMissing: false,
				},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			leftMap := internal.NewObjectMap(sep)
			for _, obj := range tc.leftObjs {
				leftMap.Add(obj)
			}
			rightMap := internal.NewObjectMap(sep)
			for _, obj := range tc.rightObjs {
				rightMap.Add(obj)
			}

			pairMap := internal.NewObjectPairMap(leftMap, rightMap)
			got := pairMap.ObjectPairs()
			require.Len(t, got, len(tc.wantPairs))

			for i, want := range tc.wantPairs {
				pair := got[i]
				assert.Equal(t, want.id, pair.ID)
				assert.Equal(t, want.isMissing, pair.IsMissing())
				if want.hasLeft {
					assert.NotNil(t, pair.Left)
					assert.Equal(t, want.id, pair.Left.Header.IntoID(sep))
				} else {
					assert.Nil(t, pair.Left)
				}
				if want.hasRight {
					assert.NotNil(t, pair.Right)
					assert.Equal(t, want.id, pair.Right.Header.IntoID(sep))
				} else {
					assert.Nil(t, pair.Right)
				}
			}
		})
	}
}
