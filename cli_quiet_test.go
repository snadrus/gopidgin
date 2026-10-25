package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestApplyQuietOnSuccess(t *testing.T) {
	// Create a tiny temp module with a .gopidgin change
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module t\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := "package p\n\nfunc H() string {\n\treturn \"x\"\n}\n"
	if err := os.WriteFile(filepath.Join(tmp, "p", "p.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	pidgin := "@ p/p.go func H\n- return \"x\"\n+ return \"y\"\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gopidgin"), []byte(pidgin), 0o644); err != nil {
		t.Fatal(err)
	}
	repoRoot, _ := os.Getwd()
	cmd := exec.Command("go", "run", ".", "apply", "-C", tmp)
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("apply should be silent on success; got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

