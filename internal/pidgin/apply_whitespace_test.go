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

func TestApplyUnified_IndentTolerance_MultiLine(t *testing.T) {
	before := "func F() {\n\tprintln(1)\n\tprintln(2)\n}\n"
	diff := []DiffLine{
		{Kind: LineContext, Text: "func F() {"},
		{Kind: LineDel, Text: "  println(1)"},
		{Kind: LineAdd, Text: "  println(10)"},
		{Kind: LineDel, Text: "\tprintln(2)"}, // mixed tab on '-' should still match
		{Kind: LineAdd, Text: "  println(20)"},
		{Kind: LineContext, Text: "}"},
	}
	after, err := ApplyUnified(before, diff)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "func F() {\n  println(10)\n  println(20)\n}\n"
	if after != want {
		t.Fatalf("got:\n%q\nwant:\n%q", after, want)
	}
}

