package internal_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
)

type repeatByteReader struct {
	b         byte
	remaining int64
}

func (r *repeatByteReader) Read(p []byte) (n int, err error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	toRead := int64(len(p))
	if toRead > r.remaining {
		toRead = r.remaining
	}
	for i := int64(0); i < toRead; i++ {
		p[i] = r.b
	}
	r.remaining -= toRead
	return int(toRead), nil
}

type yamlTestWant struct {
	I int    `yaml:"i"`
	S string `yaml:"s"`
}

func TestYamlUnmarshaler_OptionsAndLimits(t *testing.T) {
	for _, tc := range []struct {
		title                 string
		reader                io.Reader
		text                  string
		disallowDuplicateKey bool
		want                  []yamlTestWant
		wantErr               bool
		errContains           string
	}{
		{
			title: "allowDuplicateMapKey true",
			text: `---
i: 1
i: 1
s: "str"`,
			want: []yamlTestWant{
				{
					I: 1,
					S: "str",
				},
			},
		},
		{
			title: "allowDuplicateMapKey false",
			text: `---
i: 1
i: 1
s: "str"`,
			disallowDuplicateKey: true,
			wantErr:              true,
		},
		{
			title:       "size limit exceeded",
			reader:      &repeatByteReader{b: ' ', remaining: 50*1024*1024 + 10},
			wantErr:     true,
			errContains: "exceeds maximum allowed size",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			r := tc.reader
			if r == nil {
				r = strings.NewReader(tc.text)
			}
			allowDuplicateKey := !tc.disallowDuplicateKey
			got, err := internal.NewYamlUnmarshaler(r, yamlTestWant{}, allowDuplicateKey).Unmarshal(context.TODO())
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}
			if !assert.Nil(t, err) {
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestYamlUnmarshaler_Documents(t *testing.T) {
	for _, tc := range []struct {
		title string
		text  string
		want  []yamlTestWant
	}{
		{
			title: "ignore empty",
			text:  ``,
			want:  []yamlTestWant{},
		},
		{
			title: "ignore empty2",
			text: `---
`,
			want: []yamlTestWant{},
		},
		{
			title: "ignore comment only",
			text: `---
# comment`,
			want: []yamlTestWant{},
		},
		{
			title: "ignore empties",
			text: `
---
# comment
---
# comment2
---
---`,
			want: []yamlTestWant{},
		},
		{
			title: "an element",
			text: `
i: 1
s: "str"`,
			want: []yamlTestWant{
				{
					I: 1,
					S: "str",
				},
			},
		},
		{
			title: "docs",
			text: `
i: 1
s: "str"
---
i: 2
s: "str2"`,
			want: []yamlTestWant{
				{
					I: 1,
					S: "str",
				},
				{
					I: 2,
					S: "str2",
				},
			},
		},
		{
			title: "docs with empty",
			text: `
i: 1
s: "str"
---
# empty
---
i: 2
s: "str2"`,
			want: []yamlTestWant{
				{
					I: 1,
					S: "str",
				},
				{
					I: 2,
					S: "str2",
				},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got, err := internal.NewYamlUnmarshaler(
				strings.NewReader(tc.text),
				yamlTestWant{},
				true,
			).Unmarshal(context.TODO())
			if !assert.Nil(t, err) {
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestYamlMarshaler(t *testing.T) {
	for _, tc := range []struct {
		title     string
		marshaler internal.Marshaler
		obj       any
		want      string
	}{
		{
			title:     "scalar",
			marshaler: internal.NewYamlMarshaler(2, true),
			obj:       1,
			want: `1
`,
		},
		{
			title:     "dict",
			marshaler: internal.NewYamlMarshaler(2, true),
			obj: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			want: `k1: v1
k2: v2
`,
		},
		{
			title:     "array",
			marshaler: internal.NewYamlMarshaler(2, true),
			obj: []string{
				"k1",
				"k2",
			},
			want: `- k1
- k2
`,
		},
		{
			title:     "fuzz",
			marshaler: internal.NewYamlMarshaler(2, true),
			obj: []any{
				"k1",
				map[string]any{
					"k2": "v2",
					"k3": map[string]any{
						"k4": "v4",
					},
				},
			},
			want: `- k1
- k2: v2
  k3:
    k4: v4
`,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got, err := tc.marshaler.Marshal(context.TODO(), tc.obj)
			if !assert.Nil(t, err) {
				return
			}
			assert.Equal(t, tc.want, string(got))
		})
	}
}
