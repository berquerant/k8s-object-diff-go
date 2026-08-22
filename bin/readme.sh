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
README_EOF
}

generate
