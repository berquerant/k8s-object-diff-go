# k8s-object-diff-go

```
❯ objdiff --help
objdiff - k8s object diff by object id

# Usage

  objdiff [flags] LEFT_FILE RIGHT_FILE

# Object ID

A unique ID for a k8s object.
e.g.

  apiVersion: v1
  kind: Pod
  metadata:
    name: nginx
    namespace: default

then id is 'v1>Pod>default>nginx'.

# Output format
## idlist

All object IDs.

## id

ID diff.

## text

Unified diff.

## yaml

Array of

  id: "Object ID"
  diff: "Unified diff"
  left: "Left object (optional)"
  right: "Right object (optional)"

# Exit status

0 if inputs are the same.
1 if inputs differ.
Otherwise 2.

# Override differ

  objdiff -x diff left.yml right.yml
invokes
  diff --unified=3 --color=never --label left.yml --label right.yml LEFT_FILE RIGHT_FILE

  DIFFCMD='diff' objdiff -c -C 5 left.yml right.yml
invokes
  diff --unified=5 --color=always --label left.yml --label right.yml LEFT_FILE RIGHT_FILE

# Flags
      --allowDuplicateKey   allow the use of keys with the same name in the same map (default true)
  -c, --color               colored diff
  -C, --context int         diff context (default 3)
      --debug               enable debug log
  -x, --diffCmd string      invoke this to get diff instead of builtin differ
  -n, --indent int          yaml indent (default 2)
  -o, --out string          output format: text,yaml,id,idlist (default "text")
  -q, --quiet               quiet log
  -d, --separator string    object id separator (default ">")
      --success             exit with 0 even if inputs differ
      --version             print objdiff version
```

## Example

For [left.yml](./tests/diffs/left.yml) and [right.yml](./tests/diffs/right.yml), executing

``` shell
objdiff left.yml right.yml
```

yields the [result](./tests/diffs/out.txt).

## Installation

Build binary:

``` shell
make
```

Show help:

``` shell
./dist/objdiff --help
```
