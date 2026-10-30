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

// NewFieldFilter creates a [YqFilter] for removing field paths (e.g. "metadata.resourceVersion", "status")
// or direct deletion expressions (e.g. "del(.status)", "del (.status)").
func NewFieldFilter(fieldPaths []string) *YqFilter {
	if len(fieldPaths) == 0 {
		return nil
	}
	exprs := make([]string, 0, len(fieldPaths))
	for _, p := range fieldPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if isDelExpression(p) {
			exprs = append(exprs, p)
			continue
		}
		if containsDangerousYqKeyword(p) || strings.ContainsAny(p, "|;=") {
			continue
		}
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		exprs = append(exprs, fmt.Sprintf("del(%s)", p))
	}
	return NewYqFilter(exprs)
}

var dangerousYqKeywords = []string{
	"load", "load_str", "load_xml", "load_props",
	"env", "strenv", "eval", "system",
}

func containsDangerousYqKeyword(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range dangerousYqKeywords {
		if idx := strings.Index(lower, kw); idx >= 0 {
			beforeOk := idx == 0 || !isIdentChar(lower[idx-1])
			afterIdx := idx + len(kw)
			afterOk := afterIdx == len(lower) || !isIdentChar(lower[afterIdx])
			if beforeOk && afterOk {
				return true
			}
		}
	}
	return false
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

func isDelExpression(p string) bool {
	if !strings.HasSuffix(p, ")") {
		return false
	}
	var inner string
	if strings.HasPrefix(p, "del(") {
		inner = p[4 : len(p)-1]
	} else if strings.HasPrefix(p, "del") {
		rest := strings.TrimSpace(p[3:])
		if !strings.HasPrefix(rest, "(") {
			return false
		}
		inner = rest[1 : len(rest)-1]
	} else {
		return false
	}
	if strings.ContainsAny(inner, "|;=") {
		return false
	}
	return !containsDangerousYqKeyword(inner)
}

// NewLabelFilter creates a [YqFilter] for removing label keys across metadata and nested pod templates.
func NewLabelFilter(labelKeys []string) *YqFilter {
	return newMapKeyFilter("labels", labelKeys)
}

// NewAnnotationFilter creates a [YqFilter] for removing annotation keys across metadata and nested templates.
func NewAnnotationFilter(annotationKeys []string) *YqFilter {
	return newMapKeyFilter("annotations", annotationKeys)
}

func newMapKeyFilter(mapName string, keys []string) *YqFilter {
	if len(keys) == 0 {
		return nil
	}
	exprs := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if containsDangerousYqKeyword(k) || strings.ContainsAny(k, "\"\\|;=\n\r\t") {
			continue
		}
		exprs = append(exprs, fmt.Sprintf(`del(.. | .%s?."%s"?)`, mapName, escapeYqString(k)))
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
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
