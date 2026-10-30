package config_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServer(t *testing.T) {
	const (
		manifestLeft = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value1`
		manifestRight = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value2`
	)

	ctx := context.Background()
	var cfg config.Config
	server := cfg.NewMCPServer()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	sessionErrCh := make(chan error, 1)
	go func() {
		sessionErrCh <- server.Run(ctx, serverTransport)
	}()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	parseResult := func(res *mcp.CallToolResult) (config.DiffToolResult, error) {
		var toolResult config.DiffToolResult
		if res.StructuredContent != nil {
			b, err := json.Marshal(res.StructuredContent)
			if err != nil {
				return toolResult, err
			}
			if err := json.Unmarshal(b, &toolResult); err != nil {
				return toolResult, err
			}
			return toolResult, nil
		}
		if len(res.Content) > 0 {
			if textContent, ok := res.Content[0].(*mcp.TextContent); ok {
				if err := json.Unmarshal([]byte(textContent.Text), &toolResult); err != nil {
					return toolResult, err
				}
				return toolResult, nil
			}
		}
		return toolResult, nil
	}

	t.Run("list tools", func(t *testing.T) {
		tools, err := clientSession.ListTools(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, tools)
		require.Len(t, tools.Tools, 1)
		assert.Equal(t, "diff_k8s_manifests", tools.Tools[0].Name)
	})

	for _, tc := range []struct {
		name         string
		args         map[string]any
		wantDiff     bool
		wantContains []string
	}{
		{
			name: "call tool with manifest strings",
			args: map[string]any{
				"left":  manifestLeft,
				"right": manifestRight,
				"out":   "text",
			},
			wantDiff: true,
			wantContains: []string{
				"-  key: value1",
				"+  key: value2",
			},
		},
		{
			name: "call tool identical manifests",
			args: map[string]any{
				"left":  manifestLeft,
				"right": manifestLeft,
			},
			wantDiff: false,
		},
		{
			name: "call tool with file paths",
			args: map[string]any{
				"left":  filepath.Join("..", "tests", "diffs", "left.yml"),
				"right": filepath.Join("..", "tests", "diffs", "right.yml"),
				"out":   "markdown",
			},
			wantDiff: true,
			wantContains: []string{
				"Objdiff Summary",
			},
		},
		{
			name: "call tool with custom markdown title",
			args: map[string]any{
				"left":          filepath.Join("..", "tests", "diffs", "left.yml"),
				"right":         filepath.Join("..", "tests", "diffs", "right.yml"),
				"out":           "markdown",
				"markdownTitle": "Custom Title: {{ .LEFT_FILE }} <=> {{ .RIGHT_FILE }}",
			},
			wantDiff: true,
			wantContains: []string{
				"Custom Title: " + filepath.Join("..", "tests", "diffs", "left.yml") + " <=> " + filepath.Join("..", "tests", "diffs", "right.yml"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "diff_k8s_manifests",
				Arguments: tc.args,
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.False(t, res.IsError)

			toolResult, err := parseResult(res)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDiff, toolResult.HasDiff)
			if !tc.wantDiff {
				assert.Empty(t, toolResult.Diff)
			}
			for _, sub := range tc.wantContains {
				assert.Contains(t, toolResult.Diff, sub)
			}
		})
	}

	for _, tc := range []struct {
		name      string
		args      map[string]any
		wantError bool
		checkDiff func(t *testing.T, res *mcp.CallToolResult)
	}{
		{
			name: "security: reject unauthorized file path in MCP left/right",
			args: map[string]any{
				"left":  "/etc/passwd",
				"right": manifestRight,
			},
			wantError: true,
		},
		{
			name: "security: reject template file reference in MCP",
			args: map[string]any{
				"left":     manifestLeft,
				"right":    manifestRight,
				"out":      "template",
				"template": "@/etc/passwd",
			},
			wantError: true,
		},
		{
			name: "security: template env function is omitted or non-leaking",
			args: map[string]any{
				"left":     manifestLeft,
				"right":    manifestRight,
				"out":      "template",
				"template": `Result: {{ env "PATH" }}`,
			},
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "diff_k8s_manifests",
				Arguments: tc.args,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.wantError, res.IsError)
			if tc.checkDiff != nil && !res.IsError {
				tc.checkDiff(t, res)
			}
		})
	}
}

func TestMCPServerConfigImmutability(t *testing.T) {
	const (
		manifestLeft = `# comment to ignore
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value1`
		manifestRight = `# comment to ignore
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value2`
	)

	ctx := context.Background()
	baseCfg := config.Config{
		Labels:              []string{"original-left", "original-right"},
		IgnoreMatchingLines: []string{"^initial"},
		IgnoreFields:        []string{"initial.field"},
		IgnoreLabels:        []string{"initial-label"},
		IgnoreAnnotations:   []string{"initial-annotation"},
	}
	server := baseCfg.NewMCPServer()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "immutability-client",
		Version: "1.0.0",
	}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	for _, tc := range []struct {
		title string
		args  map[string]any
	}{
		{
			title: "override ignoreFields and matching lines",
			args: map[string]any{
				"left":                manifestLeft,
				"right":               manifestRight,
				"ignoreMatchingLines": []string{"^# comment"},
				"ignoreFields":        []string{"data.key"},
			},
		},
		{
			title: "override ignoreLabels and ignoreAnnotations",
			args: map[string]any{
				"left":              manifestLeft,
				"right":             manifestRight,
				"ignoreLabels":      []string{"app"},
				"ignoreAnnotations": []string{"build"},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "diff_k8s_manifests",
				Arguments: tc.args,
			})
			require.NoError(t, err)
			assert.False(t, res.IsError)

			// Base config slices must remain completely unchanged across requests
			assert.Equal(t, []string{"original-left", "original-right"}, baseCfg.Labels)
			assert.Equal(t, []string{"^initial"}, baseCfg.IgnoreMatchingLines)
			assert.Equal(t, []string{"initial.field"}, baseCfg.IgnoreFields)
			assert.Equal(t, []string{"initial-label"}, baseCfg.IgnoreLabels)
			assert.Equal(t, []string{"initial-annotation"}, baseCfg.IgnoreAnnotations)
		})
	}
}
