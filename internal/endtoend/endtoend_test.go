package endtoend

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func FindTests(t *testing.T, root string) []string {
	t.Helper()
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "sqlc.yaml" {
			dirs = append(dirs, filepath.Dir(path))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return dirs
}

func LookPath(t *testing.T, cmds ...string) string {
	t.Helper()
	for _, cmd := range cmds {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path
		}
	}
	t.Fatalf("could not find command(s) in $PATH: %s", cmds)
	return ""
}

func ExpectedOutput(t *testing.T, dir string) []byte {
	t.Helper()
	path := filepath.Join(dir, "stderr.txt")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return []byte{}
		} else {
			t.Fatal(err)
		}
	}
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func TestGenerate(t *testing.T) {
	wasmpath := filepath.Join("..", "..", "bin", "alt-sqlc-gen-python.wasm")
	if _, err := os.Stat(wasmpath); err != nil {
		t.Fatalf("alt-sqlc-gen-python.wasm not found: %s", err)
	}

	sqlc := LookPath(t, "sqlc-dev", "sqlc")

	for _, dir := range FindTests(t, "testdata") {
		dir := dir
		t.Run(dir, func(t *testing.T) {
			want := ExpectedOutput(t, dir)
			cmd := exec.Command(sqlc, "diff")
			cmd.Dir = dir
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			stdout, err := cmd.Output()
			// Filter out sha256 warnings from stderr since we omit sha256
			// from test configs to avoid updating them on every rebuild
			var filteredStderr []string
			for _, line := range strings.Split(stderr.String(), "\n") {
				if strings.Contains(line, "WARN fetching WASM binary to calculate sha256") {
					continue
				}
				filteredStderr = append(filteredStderr, line)
			}
			got := strings.Join(filteredStderr, "\n") + string(stdout)
			if diff := cmp.Diff(string(want), got); diff != "" {
				t.Errorf("sqlc diff mismatch (-want +got):\n%s", diff)
			}
			if len(want) == 0 && err != nil {
				t.Error(err)
			}
		})
	}
}
