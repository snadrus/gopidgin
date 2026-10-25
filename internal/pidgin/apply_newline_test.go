package pidgin

import "testing"

// Regression: closing brace '}' without trailing newline in base should match '-}' '+}' in diff.
func TestApplyUnified_TrailingNewlineTolerance(t *testing.T) {
	before := "func F() {\n\treturn 1\n}" // no trailing newline
	diff := []DiffLine{
		{Kind: LineContext, Text: "func F() {"},
		{Kind: LineDel, Text: "\treturn 1"},
		{Kind: LineAdd, Text: "\treturn 2"},
		{Kind: LineDel, Text: "}"},
		{Kind: LineAdd, Text: "}"},
	}
	after, err := ApplyUnified(before, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if want := "func F() {\n\treturn 2\n}\n"; after != want {
		t.Fatalf("got:\n%q\nwant:\n%q", after, want)
	}
}

