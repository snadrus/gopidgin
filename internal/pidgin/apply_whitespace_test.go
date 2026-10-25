package pidgin

import "testing"

// Proves that space-indented - lines match tab-indented file content.
func TestApplyUnified_IndentTolerance(t *testing.T) {
	before := "func Greet() string {\n\treturn \"hi\"\n}\n"
	diff := []DiffLine{
		// No context; only delete/add with leading spaces instead of tabs.
		{Kind: LineDel, Text: `  return "hi"`},
		{Kind: LineAdd, Text: `  return "hello"`},
	}
	after, err := ApplyUnified(before, diff)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "func Greet() string {\n  return \"hello\"\n}\n"
	if after != want {
		t.Fatalf("got:\n%q\nwant:\n%q", after, want)
	}
}

