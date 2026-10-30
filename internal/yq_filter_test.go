package internal_test

import (
	"testing"

	"github.com/berquerant/k8s-object-diff-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYqFilter(t *testing.T) {
	for _, tc := range []struct {
		title string
		exprs []string
	}{
		{
			title: "nil expressions",
			exprs: nil,
		},
		{
			title: "empty expressions",
			exprs: []string{},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f := internal.NewYqFilter(tc.exprs)
			got, err := f.FilterBody("apiVersion: v1\n")
			require.NoError(t, err)
			assert.Equal(t, "apiVersion: v1\n", got)
		})
	}

	const inputYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  resourceVersion: "12345"
  generation: 2
  labels:
    app: nginx
    buildId: "123"
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: "{}"
    note: "important"
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
        buildId: "123"
      annotations:
        kubectl.kubernetes.io/last-applied-configuration: "{}"
status:
  availableReplicas: 3
`

	t.Run("FieldFilter", func(t *testing.T) {
		for _, tc := range []struct {
			title       string
			fieldPaths  []string
			wantMissing []string
			wantPresent []string
		}{
			{
				title:       "without leading dot",
				fieldPaths:  []string{"metadata.resourceVersion", "status"},
				wantMissing: []string{"resourceVersion", "status"},
				wantPresent: []string{"nginx", "generation"},
			},
			{
				title:       "with leading dot",
				fieldPaths:  []string{".metadata.generation"},
				wantMissing: []string{"generation"},
				wantPresent: []string{"resourceVersion"},
			},
			{
				title:       "with explicit del() expression",
				fieldPaths:  []string{"del(.status)", "del(.metadata.resourceVersion)"},
				wantMissing: []string{"resourceVersion", "status"},
				wantPresent: []string{"nginx"},
			},
			{
				title:       "with spaces around del and parentheses",
				fieldPaths:  []string{"del (.status)", "del ( .metadata.resourceVersion )"},
				wantMissing: []string{"resourceVersion", "status"},
				wantPresent: []string{"nginx"},
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				f := internal.NewFieldFilter(tc.fieldPaths)
				require.NotNil(t, f)
				res, err := f.FilterBody(inputYAML)
				require.NoError(t, err)
				for _, m := range tc.wantMissing {
					assert.NotContains(t, res, m)
				}
				for _, p := range tc.wantPresent {
					assert.Contains(t, res, p)
				}
			})
		}
	})

	t.Run("LabelAndAnnotationFilters", func(t *testing.T) {
		for _, tc := range []struct {
			title       string
			newFilter   func([]string) *internal.YqFilter
			keys        []string
			wantMissing []string
			wantPresent []string
		}{
			{
				title:       "LabelFilter removes matching label",
				newFilter:   internal.NewLabelFilter,
				keys:        []string{"buildId"},
				wantMissing: []string{"buildId"},
				wantPresent: []string{"app: nginx"},
			},
			{
				title:       "AnnotationFilter removes matching annotation",
				newFilter:   internal.NewAnnotationFilter,
				keys:        []string{"kubectl.kubernetes.io/last-applied-configuration"},
				wantMissing: []string{"last-applied-configuration"},
				wantPresent: []string{`note: "important"`},
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				f := tc.newFilter(tc.keys)
				require.NotNil(t, f)
				res, err := f.FilterBody(inputYAML)
				require.NoError(t, err)
				for _, m := range tc.wantMissing {
					assert.NotContains(t, res, m)
				}
				for _, p := range tc.wantPresent {
					assert.Contains(t, res, p)
				}
			})
		}
	})

	t.Run("SecurityInjectionDefense", func(t *testing.T) {
		for _, tc := range []struct {
			title     string
			newFilter func([]string) *internal.YqFilter
			inputs    []string
		}{
			{
				title:     "ignore dangerous load_str in field paths",
				newFilter: internal.NewFieldFilter,
				inputs:    []string{`del(.status) | .spec.injected = load_str("/etc/passwd")`},
			},
			{
				title:     "ignore dangerous env in field paths",
				newFilter: internal.NewFieldFilter,
				inputs:    []string{`.spec | env(SECRET)`},
			},
			{
				title:     "ignore dangerous system in field paths",
				newFilter: internal.NewFieldFilter,
				inputs:    []string{`del(.status) | system("whoami")`},
			},
			{
				title:     "discard malicious injection in label keys",
				newFilter: internal.NewLabelFilter,
				inputs:    []string{`\" ) | .injected = "pwned"`},
			},
			{
				title:     "discard malicious injection in annotation keys",
				newFilter: internal.NewAnnotationFilter,
				inputs:    []string{`\" ) | .injected = "pwned"`},
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				f := tc.newFilter(tc.inputs)
				assert.Nil(t, f, "all dangerous expressions should be discarded")
			})
		}
	})
}
