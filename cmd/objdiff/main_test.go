package main_test

import (
	"bytes"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var update = flag.Bool("update", false, "update golden files")

func TestEndToEnd(t *testing.T) {
	e := newExecutor(t)
	defer e.close()

	if err := run(e.cmd, "-h"); err != nil {
		t.Fatalf("%s help %v", e.cmd, err)
	}

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
			runGoldenCase(t, e.cmd, testDir, c.Name())
		})
	}
}

type goldenTestCase struct {
	name string
	file string
}

func getGoldenTestCases(hasTemplate bool) []goldenTestCase {
	if hasTemplate {
		return []goldenTestCase{
			{name: "template", file: "out.template"},
		}
	}
	return []goldenTestCase{
		{name: "id", file: "out.id"},
		{name: "text", file: "out.txt"},
		{name: "yaml", file: "out.yml"},
		{name: "idlist", file: "out.idlist"},
		{name: "markdown", file: "out.md"},
	}
}

func runGoldenCase(t *testing.T, cmdPath, testDir, caseName string) {
	t.Helper()
	dir := filepath.Join(testDir, caseName)
	additionalArgs := parseGoldenArgs(dir)
	additionalEnvs := parseGoldenEnvs(dir)
	testCases := getGoldenTestCases(hasTemplateArg(additionalArgs))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := executeGoldenDiff(t, cmdPath, caseName, tc.name, additionalArgs, additionalEnvs)
			if !assert.Nil(t, err) {
				return
			}
			goldenFilePath := filepath.Join(dir, tc.file)
			if *update {
				if err := os.WriteFile(goldenFilePath, []byte(got), 0o644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
				return
			}
			want, err := readAllFile(goldenFilePath)
			if !assert.Nil(t, err) {
				return
			}
			assert.Equal(t, want, got)
		})
	}
}

func executeGoldenDiff(t *testing.T, cmdPath, caseName, outMode string, args, envs []string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	cmdArgs := []string{
		filepath.Join("tests", caseName, "left.yml"),
		filepath.Join("tests", caseName, "right.yml"),
		"-o", outMode, "--success",
	}
	cmdArgs = append(cmdArgs, args...)
	t.Logf("run:%v", cmdArgs)
	c := exec.Command(cmdPath, cmdArgs...)
	c.Dir = "../.."
	c.Stdout = &buf
	c.Stderr = os.Stderr
	if len(envs) > 0 {
		c.Env = append(os.Environ(), envs...)
	}
	if err := c.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func readAllFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
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

func parseGoldenArgs(dir string) []string {
	s, err := readAllFile(filepath.Join(dir, "arg.txt"))
	if err != nil {
		return nil
	}
	s = strings.TrimSpace(s)
	if strings.Contains(s, "\n") {
		var args []string
		for _, line := range strings.Split(s, "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				args = append(args, trimmed)
			}
		}
		return args
	}
	return strings.Split(s, " ")
}

func parseGoldenEnvs(dir string) []string {
	s, err := readAllFile(filepath.Join(dir, "env.txt"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(s), " ")
}

func hasTemplateArg(args []string) bool {
	for _, a := range args {
		if a == "--template" || a == "-T" || strings.HasPrefix(a, "--template=") || strings.HasPrefix(a, "-T=") {
			return true
		}
	}
	return false
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
