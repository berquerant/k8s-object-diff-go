package main_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndToEnd(t *testing.T) {
	e := newExecutor(t)
	defer e.close()

	if err := run(e.cmd, "-h"); err != nil {
		t.Fatalf("%s help %v", e.cmd, err)
	}

	//
	// Run golden tests
	//
	const testDir = "../../tests"
	cases, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}

		t.Run(c.Name(), func(t *testing.T) {
			dir := filepath.Join(testDir, c.Name())

			readAll := func(name string) (string, error) {
				f, err := os.Open(filepath.Join(dir, name))
				if err != nil {
					return "", err
				}
				defer f.Close()
				b, err := io.ReadAll(f)
				if err != nil {
					return "", err
				}
				return string(b), nil
			}

			additionalArgs := []string{}
			if s, err := readAll("arg.txt"); err == nil {
				additionalArgs = strings.Split(strings.TrimSpace(s), " ")
			}

			run := func(out string) (string, error) {
				var buf bytes.Buffer
				args := []string{
					filepath.Join("tests", c.Name(), "left.yml"),
					filepath.Join("tests", c.Name(), "right.yml"),
					"-o", out, "--success",
				}
				args = append(args, additionalArgs...)
				t.Logf("run:%v", args)
				cmd := exec.Command(e.cmd, args...)
				cmd.Dir = "../.."
				cmd.Stdout = &buf
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return "", err
				}
				return buf.String(), nil
			}

			t.Run("id", func(t *testing.T) {
				want, err := readAll("out.id")
				if !assert.Nil(t, err) {
					return
				}
				got, err := run("id")
				assert.Nil(t, err)
				assert.Equal(t, want, got)
			})
			t.Run("text", func(t *testing.T) {
				want, err := readAll("out.txt")
				if !assert.Nil(t, err) {
					return
				}
				got, err := run("text")
				assert.Nil(t, err)
				assert.Equal(t, want, got)
			})
			t.Run("yaml", func(t *testing.T) {
				want, err := readAll("out.yml")
				if !assert.Nil(t, err) {
					return
				}
				got, err := run("yaml")
				assert.Nil(t, err)
				assert.Equal(t, want, got)
			})
			t.Run("idlist", func(t *testing.T) {
				want, err := readAll("out.idlist")
				if !assert.Nil(t, err) {
					return
				}
				got, err := run("idlist")
				assert.Nil(t, err)
				assert.Equal(t, want, got)
			})
		})
	}
}

func run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type executor struct {
	dir string
	cmd string
}

func newExecutor(t *testing.T) *executor {
	t.Helper()
	e := &executor{}
	e.init(t)
	return e
}

func (e *executor) init(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "objdiff")
	if err != nil {
		t.Fatal(err)
	}
	cmd := filepath.Join(dir, "objdiff")
	// build objdiff command
	if err := run("go", "build", "-o", cmd); err != nil {
		t.Fatal(err)
	}
	e.dir = dir
	e.cmd = cmd
}

func (e *executor) close() {
	os.RemoveAll(e.dir)
}
