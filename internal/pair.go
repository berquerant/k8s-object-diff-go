package internal

import "sort"

// ObjectPair is a pair of [Objects] that share the same ID.
type ObjectPair struct {
	ID    string
	Left  *Object
	Right *Object
}

func (p *ObjectPair) IsMissing() bool {
	return p.Left == nil && p.Right == nil
}

type ObjectPairer interface {
	ObjectPairs() []*ObjectPair
}

var _ ObjectPairer = &ObjectPairMap{}

type ObjectPairMap struct {
	left  *ObjectMap
	right *ObjectMap
}

func NewObjectPairMap(left, right *ObjectMap) *ObjectPairMap {
	return &ObjectPairMap{
		left:  left,
		right: right,
	}
}

func (m *ObjectPairMap) ObjectPairs() []*ObjectPair {
	keys := m.sortedKeys()
	xs := make([]*ObjectPair, len(keys))
	for i, k := range keys {
		xs[i] = m.getObjectPair(k)
	}
	return xs
}

func (m *ObjectPairMap) sortedKeys() []string {
	s := map[string]bool{}
	for _, x := range m.left.Keys() {
		s[x] = true
	}
	for _, x := range m.right.Keys() {
		s[x] = true
	}
	ss := make([]string, len(s))
	var i int
	for k := range s {
		ss[i] = k
		i++
	}
	sort.Strings(ss)
	return ss
}

func (m *ObjectPairMap) getObjectPair(id string) *ObjectPair {
	left, _ := m.left.Get(id)
	right, _ := m.right.Get(id)
	return &ObjectPair{
		ID:    id,
		Left:  left,
		Right: right,
	}
}
