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
			require.NotNil(t, f)
			assert.True(t, f.IsEmpty())
			got, err := f.FilterBody("apiVersion: v1\n")
			require.NoError(t, err)
			assert.Equal(t, "apiVersion: v1\n", got)

			ff, err := internal.NewFieldFilter(tc.exprs)
			require.NoError(t, err)
			require.NotNil(t, ff)
			assert.True(t, ff.IsEmpty())

			lf, err := internal.NewLabelFilter(tc.exprs)
			require.NoError(t, err)
			require.NotNil(t, lf)
			assert.True(t, lf.IsEmpty())

			af, err := internal.NewAnnotationFilter(tc.exprs)
			require.NoError(t, err)
			require.NotNil(t, af)
			assert.True(t, af.IsEmpty())
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
    env: production
    system: core
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: "{}"
    note: "important"
    env: annotation-env-val
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
        buildId: "123"
        env: production
      annotations:
        kubectl.kubernetes.io/last-applied-configuration: "{}"
    spec:
      containers:
        - name: app
          image: nginx:latest
          env:
            - name: FOO
              value: bar
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
			{
				title:       "allow legitimate kubernetes field named env",
				fieldPaths:  []string{"spec.template.spec.containers[].env"},
				wantMissing: []string{"FOO", "bar"},
				wantPresent: []string{"nginx:latest"},
			},
			{
				title:       "allow array traversal field path",
				fieldPaths:  []string{"spec.template.spec.containers[].image"},
				wantMissing: []string{"nginx:latest"},
				wantPresent: []string{"FOO"},
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				f, err := internal.NewFieldFilter(tc.fieldPaths)
				require.NoError(t, err)
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
			newFilter   func([]string) (*internal.YqFilter, error)
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
				title:       "LabelFilter removes legitimate env and system labels",
				newFilter:   internal.NewLabelFilter,
				keys:        []string{"env", "system"},
				wantMissing: []string{"production", "core"},
				wantPresent: []string{"app: nginx"},
			},
			{
				title:       "AnnotationFilter removes matching annotation",
				newFilter:   internal.NewAnnotationFilter,
				keys:        []string{"kubectl.kubernetes.io/last-applied-configuration"},
				wantMissing: []string{"last-applied-configuration"},
				wantPresent: []string{`note: "important"`},
			},
			{
				title:       "AnnotationFilter removes legitimate env annotation",
				newFilter:   internal.NewAnnotationFilter,
				keys:        []string{"env"},
				wantMissing: []string{"annotation-env-val"},
				wantPresent: []string{`note: "important"`, "env: production"},
			},
		} {
			t.Run(tc.title, func(t *testing.T) {
				f, err := tc.newFilter(tc.keys)
				require.NoError(t, err)
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
}

func TestYqFilter_SecurityInjectionDefense(t *testing.T) {
	for _, tc := range []struct {
		title     string
		newFilter func([]string) (*internal.YqFilter, error)
		inputs    []string
	}{
		{
			title:     "reject dangerous load_str in field paths",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`del(.status) | .spec.injected = load_str("/etc/passwd")`},
		},
		{
			title:     "reject load_base64 via comma union bypass",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`del(.status), load_base64("/etc/passwd")`},
		},
		{
			title:     "reject load_base64 via closing paren breakout",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`status), load_base64("/etc/passwd")`},
		},
		{
			title:     "reject envsubst via comma union bypass",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`del(.status), envsubst`},
		},
		{
			title:     "reject envsubst via closing paren breakout",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`status), envsubst`},
		},
		{
			title:     "reject dangerous env in field paths",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`.spec | env(SECRET)`},
		},
		{
			title:     "reject dangerous env function call inside path",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`spec.environment.env(SECRET)`},
		},
		{
			title:     "reject dangerous system in field paths",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`del(.status) | system("whoami")`},
		},
		{
			title:     "reject assignment injection in field paths",
			newFilter: internal.NewFieldFilter,
			inputs:    []string{`.spec.replicas = 999`},
		},
		{
			title:     "reject malicious injection in label keys",
			newFilter: internal.NewLabelFilter,
			inputs:    []string{`\" ) | .injected = "pwned"`},
		},
		{
			title:     "reject malicious injection in annotation keys",
			newFilter: internal.NewAnnotationFilter,
			inputs:    []string{`\" ) | .injected = "pwned"`},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f, err := tc.newFilter(tc.inputs)
			require.Error(t, err, "dangerous expression must return an error")
			assert.Nil(t, f)
		})
	}

	t.Run("engine-level security preferences block file/env/system ops even via NewYqFilter", func(t *testing.T) {
		for _, expr := range []string{
			`load_str("/etc/passwd")`,
			`load_base64("/etc/passwd")`,
			`env(PATH)`,
			`strenv(PATH)`,
			`envsubst`,
			`system("echo hi")`,
		} {
			t.Run(expr, func(t *testing.T) {
				f := internal.NewYqFilter([]string{expr})
				require.NotNil(t, f)
				_, err := f.FilterBody("a: \"${PATH}\"\n")
				assert.Error(t, err)
			})
		}
	})

	for _, tc := range []struct {
		title string
		input string
	}{
		{
			title: "allow safe word containing keyword substring environment",
			input: "spec.environment",
		},
		{
			title: "allow safe word containing keyword substring subsystem_metric",
			input: "metadata.subsystem_metric",
		},
		{
			title: "allow safe field named env",
			input: "spec.template.spec.containers[].env",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f, err := internal.NewFieldFilter([]string{tc.input})
			require.NoError(t, err)
			assert.NotNil(t, f, "safe field path should be allowed")
		})
	}
}
