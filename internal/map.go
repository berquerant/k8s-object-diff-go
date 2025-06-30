package internal

type ObjectMap struct {
	d   map[string]*Object
	sep string
}

func NewObjectMap(sep string) *ObjectMap {
	return &ObjectMap{
		d:   map[string]*Object{},
		sep: sep,
	}
}

func (m *ObjectMap) Add(obj *Object) bool {
	id := obj.Header.IntoID(m.sep)
	var exist bool
	_, exist = m.d[id]
	m.d[id] = obj
	return exist
}

func (m *ObjectMap) Get(id string) (*Object, bool) {
	v, ok := m.d[id]
	return v, ok
}

func (m *ObjectMap) Keys() []string {
	keys := make([]string, len(m.d))
	var i int
	for k := range m.d {
		keys[i] = k
		i++
	}
	return keys
}
