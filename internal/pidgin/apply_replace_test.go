package pidgin

import "testing"

func TestApplyUnified_WholeSpanReplaceOnlyAdds(t *testing.T) {
	before := "func Hello() string {\n\treturn \"a\"\n}\n"
	diff := []DiffLine{
		{Kind: LineAdd, Text: "func Hello() string {"},
		{Kind: LineAdd, Text: "\treturn \"b\""},
		{Kind: LineAdd, Text: "}"},
	}
	after, err := ApplyUnified(before, diff)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "func Hello() string {\n\treturn \"b\"\n}\n"
	if after != want {
		t.Fatalf("got:\n%q\nwant:\n%q", after, want)
	}
}

func TestApplyUnified_OnlyAddsEmptyErrors(t *testing.T) {
	before := "x\n"
	_, err := ApplyUnified(before, nil)
	if err == nil {
		t.Fatalf("expected error for empty replacement")
	}
}

