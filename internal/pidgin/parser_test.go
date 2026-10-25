package pidgin

import (
	"strings"
	"testing"
)

func TestParseHeaderAndDiff(t *testing.T) {
	src := `
@ foo/bar.go func Hello
  func Hello() {
-   println("old")
+   println("new")
  }
`
	h, err := Parse(src)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if len(h) != 1 {
		t.Fatalf("want 1 hunk, got %d", len(h))
	}
	got := h[0]
	if got.RelPath != "foo/bar.go" || got.Action != ActionModify {
		t.Fatalf("header parse mismatch: %+v", got)
	}
	if got.Anchor.Kind != AnchorFunc || got.Anchor.Name != "Hello" {
		t.Fatalf("anchor parse mismatch: %+v", got.Anchor)
	}
	if len(got.Diff) != 4 {
		t.Fatalf("diff lines = %d", len(got.Diff))
	}
}

func TestAliasExpansion(t *testing.T) {
	src := `
@ pkg/file.go func Hello
$name=World
  fmt.Println($name)
`
	h, err := Parse(src)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if got, want := h[0].Diff[0].Text, " fmt.Println(World)"; got != want {
		t.Fatalf("alias expand: got %q want %q", got, want)
	}
}

func TestSpecialImportsParse(t *testing.T) {
	src := `
@ pkg/file.go func Hello
+ import sql "database/sql"
+ import _ "image/png"
+ import . "fmt"
  // body
`
	h, err := Parse(src)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	imps := h[0].SpecialImportsAdd
	if len(imps) != 3 {
		t.Fatalf("want 3 special imports, got %d", len(imps))
	}
}

func TestApplyUnified(t *testing.T) {
	before := strings.TrimSpace(`
func Hello() {
	println("old")
}
`) + "\n"
	diff := []DiffLine{
		{Kind: LineContext, Text: "func Hello() {"},
		{Kind: LineDel, Text: `	println("old")`},
		{Kind: LineAdd, Text: `	println("new")`},
		{Kind: LineContext, Text: "}"},
	}
	after, err := ApplyUnified(before, diff)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := strings.TrimSpace(`
func Hello() {
	println("new")
}
`) + "\n"
	if after != want {
		t.Fatalf("got:\n%s\nwant:\n%s", after, want)
	}
}

