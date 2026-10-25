package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestApplyReadsFromStdinWhenNoFileArg(t *testing.T) {
	// Create a tiny temp module
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module tempmod\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "package p\n\nfunc H() string {\n\treturn \"a\"\n}\n"
	if err := os.MkdirAll(filepath.Join(tmp, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "p", "p.go")
	if err := os.WriteFile(target, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pidgin doc to change return
	pidgin := "@ p/p.go func H\n- return \"a\"\n+ return \"b\"\n"
	// Run the CLI from repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "apply", "-C", tmp)
	cmd.Dir = repoRoot
	cmd.Stdin = bytes.NewBufferString(pidgin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("apply failed: %v\n%s", err, string(out))
	}
	// Verify change
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(after, []byte(`return "b"`)) {
		t.Fatalf("did not apply change:\n%s", string(after))
	}
}

func TestApplyReadsFromDashArg(t *testing.T) {
	// Create a tiny temp module
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module tempmod\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "package p\n\nfunc H() string {\n\treturn \"a\"\n}\n"
	if err := os.MkdirAll(filepath.Join(tmp, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "p", "p.go")
	if err := os.WriteFile(target, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	pidgin := "@ p/p.go func H\n- return \"a\"\n+ return \"c\"\n"
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "apply", "-C", tmp, "-")
	cmd.Dir = repoRoot
	cmd.Stdin = bytes.NewBufferString(pidgin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("apply failed: %v\n%s", err, string(out))
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(after, []byte(`return "c"`)) {
		t.Fatalf("did not apply change:\n%s", string(after))
	}
}

