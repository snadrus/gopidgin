package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestApplyCreateTxtViaStdin_NoGoFormatting(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module t\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pidgin := "@ NOTES.txt +\nHello\n"
	repoRoot, _ := os.Getwd()
	cmd := exec.Command("go", "run", ".", "apply", "-C", tmp)
	cmd.Dir = repoRoot
	cmd.Stdin = bytes.NewBufferString(pidgin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("apply failed: %v\n%s", err, string(out))
	}
	data, err := os.ReadFile(filepath.Join(tmp, "NOTES.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Hello\n" && string(data) != "Hello" {
		t.Fatalf("unexpected contents: %q", string(data))
	}
}

func TestApplyMultiHunk_Stdin_GoAndTxt(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module t\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	goSrc := "package p\n\nfunc H() string {\n\treturn \"a\"\n}\n"
	if err := os.WriteFile(filepath.Join(tmp, "p", "p.go"), []byte(goSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	// Two hunks: modify Go (requires fmt) and create txt
	pidgin := "" +
		"@ p/p.go func H\n" +
		"- return \"a\"\n" +
		"+ return fmt.Sprint(1)\n" +
		"@ DOC.txt +\n" +
		"doc\n"
	repoRoot, _ := os.Getwd()
	cmd := exec.Command("go", "run", ".", "apply", "-C", tmp)
	cmd.Dir = repoRoot
	cmd.Stdin = bytes.NewBufferString(pidgin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("apply failed: %v\n%s", err, string(out))
	}
	// Go file formatted and imports added
	godata, err := os.ReadFile(filepath.Join(tmp, "p", "p.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(godata, []byte(`"fmt"`)) || !bytes.Contains(godata, []byte(`return fmt.Sprint(1)`)) {
		t.Fatalf("go file not formatted/imported:\n%s", string(godata))
	}
	// Non-go created raw
	txt, err := os.ReadFile(filepath.Join(tmp, "DOC.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(txt) != "doc\n" && string(txt) != "doc" {
		t.Fatalf("unexpected txt content: %q", string(txt))
	}
}

