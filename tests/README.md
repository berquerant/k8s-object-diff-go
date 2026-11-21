# tests

This is the location of the golden test.

## Update tests

At the root directory of the project,

``` shell
make golden
```

## Add test

1. add a new directory like `tests/new-test`
2. add `left.yml` and `right.yml`
3. (optional) add `arg.txt` for additional arguments of `objdiff` (newline-delimited when multi-line; space-separated on a single line)
4. (optional) add `env.txt` for space-separated environment variables passed to `objdiff`
5. (optional) add `template.tmpl` and reference it via `-T` / `--template` in `arg.txt` to generate `out.template`
6. run `make golden` (`Update tests`)
