#!/bin/bash

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly root="$(cd "${d}/.." || exit 1 ; pwd)"

generate() {
    cd "${root}"
    local tmp_bin
    tmp_bin="$(mktemp -d)/objdiff"
    go build -o "${tmp_bin}" ./cmd/objdiff
    local help_out
    help_out="$("${tmp_bin}" --help 2>&1)"
    rm -rf "$(dirname "${tmp_bin}")"

    cat << README_EOF
# k8s-object-diff-go

${help_out}

## Example

For [left.yml](./tests/diffs/left.yml) and [right.yml](./tests/diffs/right.yml), executing

\`\`\` shell
objdiff left.yml right.yml
\`\`\`

yields the [result](./tests/diffs/out.txt).

## Installation

\`\`\` shell
go install github.com/berquerant/k8s-object-diff-go/cmd/objdiff@latest
\`\`\`

## MCP Server Configuration

You can register \`objdiff\` as a Model Context Protocol (MCP) server for AI agents using \`--mcp\`.

### MCP Settings JSON

\`\`\`json
{
  "mcpServers": {
    "objdiff": {
      "command": "objdiff",
      "args": ["--mcp"]
    }
  }
}
\`\`\`

### Exposed Tool: \`diff_k8s_manifests\`

Compares two Kubernetes manifests by Object ID and returns \`{ "diff": string, "hasDiff": bool }\`.

Key parameters:
- \`left\`, \`right\` (required): Inline Kubernetes manifest YAML strings (filesystem paths are not opened).
- \`out\`: Output format (\`text\`, \`yaml\`, \`id\`, \`idlist\`, \`markdown\`, \`template\`).
- \`context\`, \`indent\`, \`separator\`, \`verbose\`, \`markdownHeadingLevel\` (1-6), \`markdownTitle\`, \`template\`.
- \`ignoreMatchingLines\`, \`ignoreFields\`, \`ignoreLabels\`, \`ignoreAnnotations\`, \`ignoreManagedFields\`, \`ignoreStatus\`.

Security constraints in MCP mode:
- \`left\` and \`right\` accept inline YAML content only (max 50MB per input).
- Template file references (\`@file\`) are disabled; pass inline templates only.
- \`yq\` filters run with environment, file, and system operations disabled and only allow safe \`del(...)\` path expressions.
README_EOF
}

generate
