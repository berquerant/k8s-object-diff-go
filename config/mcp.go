package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DiffToolParams defines the input parameters for the diff_k8s_manifests tool.
type DiffToolParams struct {
	Left                 string   `json:"left" jsonschema:"left manifest YAML content or file path"`
	Right                string   `json:"right" jsonschema:"right manifest YAML content or file path"`
	Out                  string   `json:"out,omitempty" jsonschema:"output format: text, yaml, id, idlist, markdown (default: text)"`
	Context              *int     `json:"context,omitempty" jsonschema:"number of diff context lines (default: 3)"`
	Separator            string   `json:"separator,omitempty" jsonschema:"object id separator (default: >)"`
	Indent               *int     `json:"indent,omitempty" jsonschema:"yaml indent (default: 2)"`
	Verbose              bool     `json:"verbose,omitempty" jsonschema:"enable verbose output; annotate diff type and display summary"`
	MarkdownHeadingLevel *uint    `json:"markdownHeadingLevel,omitempty" jsonschema:"highest heading level in markdown (default: 1)"`
	IgnoreMatchingLines  []string `json:"ignoreMatchingLines,omitempty" jsonschema:"ignore lines matching regexp"`
	IgnoreFields         []string `json:"ignoreFields,omitempty" jsonschema:"ignore field by path or yq expression"`
	IgnoreLabels         []string `json:"ignoreLabels,omitempty" jsonschema:"ignore label by key"`
	IgnoreAnnotations    []string `json:"ignoreAnnotations,omitempty" jsonschema:"ignore annotation by key"`
	IgnoreManagedFields  bool     `json:"ignoreManagedFields,omitempty" jsonschema:"ignore metadata.managedFields"`
	IgnoreStatus         bool     `json:"ignoreStatus,omitempty" jsonschema:"ignore status field"`
}

// DiffToolResult defines the output of the diff_k8s_manifests tool.
type DiffToolResult struct {
	Diff    string `json:"diff" jsonschema:"the formatted diff output"`
	HasDiff bool   `json:"hasDiff" jsonschema:"true if differences were found"`
}

// NewMCPServer creates and returns a configured MCP Server for objdiff.
func (c *Config) NewMCPServer() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "objdiff",
			Version: version.Version,
		},
		nil,
	)

	tool := &mcp.Tool{
		Name:        "diff_k8s_manifests",
		Description: "Compare Kubernetes manifests by object ID and return structured diff results in text, markdown, or yaml format.",
	}

	mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, params DiffToolParams) (*mcp.CallToolResult, DiffToolResult, error) {
		res, err := c.runDiffTool(ctx, params)
		if err != nil {
			return nil, DiffToolResult{}, err
		}
		return nil, res, nil
	})

	return s
}

// RunMCP starts the MCP server over stdio.
func (c *Config) RunMCP(ctx context.Context) error {
	s := c.NewMCPServer()
	t := &mcp.StdioTransport{}
	return s.Run(ctx, t)
}

func (c *Config) runDiffTool(ctx context.Context, params DiffToolParams) (DiffToolResult, error) {
	// Create a copy of config and apply param overrides
	cfg := *c

	if cfg.Separator == "" {
		cfg.Separator = ">"
	}
	if cfg.Indent == 0 {
		cfg.Indent = 2
	}
	if params.Out != "" {
		cfg.Out = params.Out
	}
	if params.Context != nil {
		cfg.Context = *params.Context
	} else if cfg.Context == 0 {
		cfg.Context = 3
	}
	if params.Separator != "" {
		cfg.Separator = params.Separator
	}
	if params.Indent != nil {
		cfg.Indent = *params.Indent
	}
	if params.Verbose {
		cfg.Verbose = true
	}
	if params.MarkdownHeadingLevel != nil {
		cfg.MarkdownHeadingLevel = *params.MarkdownHeadingLevel
	}
	if len(params.IgnoreMatchingLines) > 0 {
		cfg.IgnoreMatchingLines = append(cfg.IgnoreMatchingLines, params.IgnoreMatchingLines...)
	}
	if len(params.IgnoreFields) > 0 {
		cfg.IgnoreFields = append(cfg.IgnoreFields, params.IgnoreFields...)
	}
	if len(params.IgnoreLabels) > 0 {
		cfg.IgnoreLabels = append(cfg.IgnoreLabels, params.IgnoreLabels...)
	}
	if len(params.IgnoreAnnotations) > 0 {
		cfg.IgnoreAnnotations = append(cfg.IgnoreAnnotations, params.IgnoreAnnotations...)
	}
	if params.IgnoreManagedFields {
		cfg.IgnoreManagedFields = true
	}
	if params.IgnoreStatus {
		cfg.IgnoreStatus = true
	}

	var leftReader io.Reader
	leftLabel := "left"
	if isFile(params.Left) {
		f, err := os.Open(params.Left)
		if err != nil {
			return DiffToolResult{}, fmt.Errorf("open left file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()
		leftReader = f
		leftLabel = params.Left
	} else {
		leftReader = strings.NewReader(params.Left)
	}

	var rightReader io.Reader
	rightLabel := "right"
	if isFile(params.Right) {
		f, err := os.Open(params.Right)
		if err != nil {
			return DiffToolResult{}, fmt.Errorf("open right file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()
		rightReader = f
		rightLabel = params.Right
	} else {
		rightReader = strings.NewReader(params.Right)
	}

	if len(cfg.Labels) == 0 {
		cfg.Labels = []string{leftLabel, rightLabel}
	}

	var buf bytes.Buffer
	err := cfg.runWithReaders(ctx, &buf, leftReader, rightReader)
	hasDiff := false
	if err != nil {
		if errors.Is(err, ErrDiffFound) {
			hasDiff = true
		} else {
			return DiffToolResult{}, err
		}
	}

	return DiffToolResult{
		Diff:    buf.String(),
		HasDiff: hasDiff,
	}, nil
}

func isFile(pathOrContent string) bool {
	if strings.Contains(pathOrContent, "\n") {
		return false
	}
	info, err := os.Stat(pathOrContent)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
