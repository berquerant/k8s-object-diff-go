package internal

import (
	"fmt"
	"strings"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
)

func init() {
	// Enforce engine-level sandboxing for all yq evaluations:
	// - DisableEnvOps blocks env(), strenv(), and envsubst from reading host environment variables.
	// - DisableFileOps blocks load(), load_str(), load_base64(), etc. from reading local files.
	// - EnableSystemOps=false keeps external command execution (system()) disabled.
	yqlib.ConfiguredSecurityPreferences.DisableEnvOps = true
	yqlib.ConfiguredSecurityPreferences.DisableFileOps = true
	yqlib.ConfiguredSecurityPreferences.EnableSystemOps = false
	yqlib.InitExpressionParser()
}

// YqFilter evaluates yq deletion/transformation expressions on YAML content.
type YqFilter struct {
	expression string
	evaluator  yqlib.StringEvaluator
}

// NewYqFilter creates a [YqFilter] with the given raw yq expressions piped together.
// Returns a no-op filter (IsEmpty() == true) if no non-empty expressions are provided.
func NewYqFilter(exprs []string) *YqFilter {
	valid := make([]string, 0, len(exprs))
	for _, e := range exprs {
		if trimmed := strings.TrimSpace(e); trimmed != "" {
			valid = append(valid, trimmed)
		}
	}
	if len(valid) == 0 {
		return &YqFilter{}
	}
	return &YqFilter{
		expression: strings.Join(valid, " | "),
		evaluator:  yqlib.NewStringEvaluator(),
	}
}

// IsEmpty reports whether the filter has no expressions to evaluate (no-op).
func (f *YqFilter) IsEmpty() bool {
	return f == nil || f.expression == ""
}

// allowedDeletePathOps lists yq AST operation types permitted inside a user-supplied del(...) target.
var allowedDeletePathOps = map[string]bool{
	"TRAVERSE_PATH":     true, // field access (e.g. .metadata, .status, ."foo")
	"TRAVERSE_ARRAY":    true, // array/map indexing (e.g. [0], [], ["key"])
	"SHORT_PIPE":        true, // internal chaining of path/array segments (e.g. .spec.containers[].image)
	"VALUE":             true, // literal values used as indices or keys (e.g. 0, "foo")
	"COLLECT":           true, // array iteration internal representation (e.g. [])
	"SELF":              true, // current/root node reference (.)
	"EMPTY":             true, // empty expression/bracket placeholder
	"RECURSIVE_DESCENT": true, // recursive descent operator (..)
}

// NewFieldFilter creates a [YqFilter] for removing field paths (e.g. "metadata.resourceVersion", "status")
// or direct deletion expressions (e.g. "del(.status)", "del (.status)").
// Returns a no-op filter if fieldPaths contains no non-empty entries, or an error if any expression is invalid or disallowed.
func NewFieldFilter(fieldPaths []string) (*YqFilter, error) {
	if len(fieldPaths) == 0 {
		return &YqFilter{}, nil
	}
	exprs := make([]string, 0, len(fieldPaths))
	for _, p := range fieldPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		expr := normalizeFieldDeleteExpr(p)
		if err := validateDeleteExpression(expr); err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}
	return NewYqFilter(exprs), nil
}

func normalizeFieldDeleteExpr(p string) string {
	if isDelWrapper(p) {
		return p
	}
	if !strings.HasPrefix(p, ".") {
		p = "." + p
	}
	return fmt.Sprintf("del(%s)", p)
}

func isDelWrapper(p string) bool {
	if !strings.HasSuffix(p, ")") || !strings.HasPrefix(p, "del") {
		return false
	}
	rest := strings.TrimSpace(p[3:])
	return strings.HasPrefix(rest, "(")
}

func validateDeleteExpression(expr string) error {
	node, err := yqlib.ExpressionParser.ParseExpression(expr)
	if err != nil {
		return fmt.Errorf("invalid yq expression %q: %w", expr, err)
	}
	if node == nil || node.Operation == nil || node.Operation.OperationType == nil || node.Operation.OperationType.Type != "DELETE" {
		return fmt.Errorf("only single del(...) expressions are allowed: %q", expr)
	}
	if node.LHS != nil {
		return fmt.Errorf("unexpected left-hand side in del(...) expression: %q", expr)
	}
	if node.RHS == nil {
		return fmt.Errorf("empty target in del(...) expression: %q", expr)
	}
	return validateDeleteTargetNode(node.RHS, expr)
}

func validateDeleteTargetNode(node *yqlib.ExpressionNode, expr string) error {
	if node == nil {
		return nil
	}
	if node.Operation == nil || node.Operation.OperationType == nil {
		return fmt.Errorf("invalid operation in yq expression %q", expr)
	}
	opType := node.Operation.OperationType.Type
	if !allowedDeletePathOps[opType] {
		return fmt.Errorf("disallowed yq operation %q in expression %q", opType, expr)
	}
	if err := validateDeleteTargetNode(node.LHS, expr); err != nil {
		return err
	}
	return validateDeleteTargetNode(node.RHS, expr)
}

// NewLabelFilter creates a [YqFilter] for removing label keys across metadata and nested pod templates.
func NewLabelFilter(labelKeys []string) (*YqFilter, error) {
	return newMapKeyFilter("labels", labelKeys)
}

// NewAnnotationFilter creates a [YqFilter] for removing annotation keys across metadata and nested templates.
func NewAnnotationFilter(annotationKeys []string) (*YqFilter, error) {
	return newMapKeyFilter("annotations", annotationKeys)
}

func newMapKeyFilter(mapName string, keys []string) (*YqFilter, error) {
	if len(keys) == 0 {
		return &YqFilter{}, nil
	}
	exprs := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if strings.ContainsAny(k, "\"\\\x00\n\r\t") {
			return nil, fmt.Errorf("invalid %s key %q", mapName, k)
		}
		expr := fmt.Sprintf(`del(.. | .%s?."%s"?)`, mapName, escapeYqString(k))
		if _, err := yqlib.ExpressionParser.ParseExpression(expr); err != nil {
			return nil, fmt.Errorf("invalid %s key %q: %w", mapName, k, err)
		}
		exprs = append(exprs, expr)
	}
	return NewYqFilter(exprs), nil
}

// FilterBody evaluates the yq expression on the given YAML string and returns the filtered YAML string.
func (f *YqFilter) FilterBody(body string) (string, error) {
	if f == nil || f.expression == "" || strings.TrimSpace(body) == "" {
		return body, nil
	}
	prefs := yqlib.ConfiguredYamlPreferences.Copy()
	encoder := yqlib.NewYamlEncoder(prefs)
	decoder := yqlib.NewYamlDecoder(prefs)

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
