#!/bin/bash

set -euo pipefail

short_sha() {
    git rev-parse --short HEAD
}

current_tag() {
    git describe --tags --abbrev=0 --exact-match 2>/dev/null || true
}

readonly version_package="github.com/berquerant/k8s-object-diff-go/version"

ldflags() {
    local flags="-X ${version_package}.Revision=$(short_sha)"
    local tag
    tag="$(current_tag)"
    if [ -n "${tag}" ]; then
        flags="-X ${version_package}.Version=${tag} ${flags}"
    fi
    echo "${flags}"
}

build() {
    go build -v -trimpath -ldflags "$(ldflags)" "$@"
}

build "$@"
