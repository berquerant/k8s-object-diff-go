package internal

import (
	"fmt"
	"strings"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
)

func init() {
	yqlib.InitExpressionParser()
}

// FieldFilter removes specified field paths from YAML content using yqlib.
type FieldFilter struct {
	expression string
	evaluator  yqlib.StringEvaluator
}

// NewFieldFilter creates a [FieldFilter] given a slice of field paths (e.g. "metadata.resourceVersion", "status").
func NewFieldFilter(fieldPaths []string) *FieldFilter {
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
	return &FieldFilter{
		expression: strings.Join(exprs, " | "),
		evaluator:  yqlib.NewStringEvaluator(),
	}
}

// FilterBody evaluates the yq deletion expression on the given YAML string.
func (f *FieldFilter) FilterBody(body string) (string, error) {
	if f == nil || f.expression == "" || strings.TrimSpace(body) == "" {
		return body, nil
	}
	encoder := yqlib.NewYamlEncoder(yqlib.ConfiguredYamlPreferences)
	decoder := yqlib.NewYamlDecoder(yqlib.ConfiguredYamlPreferences)

	res, err := f.evaluator.EvaluateAll(f.expression, body, encoder, decoder)
	if err != nil {
		return "", fmt.Errorf("field filter: %w", err)
	}
	return res, nil
}
