#!/bin/bash

#
# Generate goldens
#

run() {
    local -r _target="$1"
    local -r _out="$2"
    echo >&2 "Build golden ${_target} out=${_out}"
    if [[ -s "${_target}/arg.txt" ]] ; then
        go run ./cmd/objdiff "${_target}/left.yml" "${_target}/right.yml" -o "${_out}" $(cat "${_target}/arg.txt"|tr '\n' " ") 2>/dev/null
    else
            go run ./cmd/objdiff "${_target}/left.yml" "${_target}/right.yml" -o "${_out}" 2>/dev/null
    fi
}

list() {
    ls -1 tests/ | grep -v "README.md" | awk '{print "tests/"$1}'
}

list | while read -r target ; do
    # for output id
    run "$target" id > "${target}/out.id"
    # for output text
    run "$target" text > "${target}/out.txt"
    # for output yaml
    run "$target" yaml > "${target}/out.yml"
    # for output idlist
    run "$target" idlist > "${target}/out.idlist"
done

echo >&2 "End golden"
