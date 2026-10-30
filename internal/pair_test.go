package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expectedPair struct {
	id        string
	hasLeft   bool
	hasRight  bool
	isMissing bool
}

func newTestObj(apiVersion, kind, namespace, name string) *internal.Object {
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

func verifyObjectPairs(t *testing.T, sep string, leftObjs, rightObjs []*internal.Object, wantPairs []expectedPair) {
	t.Helper()
	leftMap := internal.NewObjectMap(sep)
	for _, obj := range leftObjs {
		leftMap.Add(obj)
	}
	rightMap := internal.NewObjectMap(sep)
	for _, obj := range rightObjs {
		rightMap.Add(obj)
	}

	pairMap := internal.NewObjectPairMap(leftMap, rightMap)
	got := pairMap.ObjectPairs()
	require.Len(t, got, len(wantPairs))

	for i, want := range wantPairs {
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
}

func TestObjectPair_SingleSide(t *testing.T) {
	const sep = ">"
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
				newTestObj("v1", "Pod", "default", "p1"),
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
				newTestObj("v1", "Pod", "default", "p2"),
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
	} {
		t.Run(tc.title, func(t *testing.T) {
			verifyObjectPairs(t, sep, tc.leftObjs, tc.rightObjs, tc.wantPairs)
		})
	}
}

func TestObjectPair_Mixed(t *testing.T) {
	const sep = ">"
	tc := struct {
		title     string
		leftObjs  []*internal.Object
		rightObjs []*internal.Object
		wantPairs []expectedPair
	}{
		title: "mixed pairs sorted by object ID",
		leftObjs: []*internal.Object{
			newTestObj("v1", "k1", "default", "n1"),
			newTestObj("v1", "k1", "default", "n2"),
		},
		rightObjs: []*internal.Object{
			newTestObj("v1", "k1", "default", "n2"),
			newTestObj("v1", "k1", "default", "n3"),
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
	}

	t.Run(tc.title, func(t *testing.T) {
		verifyObjectPairs(t, sep, tc.leftObjs, tc.rightObjs, tc.wantPairs)
	})
}
