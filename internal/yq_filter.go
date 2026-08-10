package internal

import (
	"fmt"
	"strings"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
)

func init() {
	yqlib.InitExpressionParser()
}

// YqFilter evaluates yq deletion/transformation expressions on YAML content.
type YqFilter struct {
	expression string
	evaluator  yqlib.StringEvaluator
}

// NewYqFilter creates a [YqFilter] with the given raw yq expressions piped together.
func NewYqFilter(exprs []string) *YqFilter {
	valid := make([]string, 0, len(exprs))
	for _, e := range exprs {
		if trimmed := strings.TrimSpace(e); trimmed != "" {
			valid = append(valid, trimmed)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	return &YqFilter{
		expression: strings.Join(valid, " | "),
		evaluator:  yqlib.NewStringEvaluator(),
	}
}

// NewFieldFilter creates a [YqFilter] for removing field paths (e.g. "metadata.resourceVersion", "status").
func NewFieldFilter(fieldPaths []string) *YqFilter {
	if len(fieldPaths) == 0 {
		return nil
	}
	exprs := make([]string, len(fieldPaths))
	for i, p := range fieldPaths {
		p = strings.TrimSpace(p)
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		exprs[i] = fmt.Sprintf("del(%s)", p)
	}
	return NewYqFilter(exprs)
}

// NewLabelFilter creates a [YqFilter] for removing label keys across metadata and nested pod templates.
func NewLabelFilter(labelKeys []string) *YqFilter {
	if len(labelKeys) == 0 {
		return nil
	}
	exprs := make([]string, len(labelKeys))
	for i, k := range labelKeys {
		k = strings.TrimSpace(k)
		exprs[i] = fmt.Sprintf(`del(.. | .labels?."%s"?)`, escapeYqString(k))
	}
	return NewYqFilter(exprs)
}

// NewAnnotationFilter creates a [YqFilter] for removing annotation keys across metadata and nested templates.
func NewAnnotationFilter(annotationKeys []string) *YqFilter {
	if len(annotationKeys) == 0 {
		return nil
	}
	exprs := make([]string, len(annotationKeys))
	for i, k := range annotationKeys {
		k = strings.TrimSpace(k)
		exprs[i] = fmt.Sprintf(`del(.. | .annotations?."%s"?)`, escapeYqString(k))
	}
	return NewYqFilter(exprs)
}

// FilterBody evaluates the yq expression on the given YAML string and returns the filtered YAML string.
func (f *YqFilter) FilterBody(body string) (string, error) {
	if f == nil || f.expression == "" || strings.TrimSpace(body) == "" {
		return body, nil
	}
	encoder := yqlib.NewYamlEncoder(yqlib.ConfiguredYamlPreferences)
	decoder := yqlib.NewYamlDecoder(yqlib.ConfiguredYamlPreferences)

	res, err := f.evaluator.EvaluateAll(f.expression, body, encoder, decoder)
	if err != nil {
		return "", fmt.Errorf("yq filter evaluate (%s): %w", f.expression, err)
	}
	return res, nil
}

func escapeYqString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
