package pidgin

import "testing"

func TestApplyUnified_MismatchError(t *testing.T) {
	before := "func F(){\n\tprintln(\"x\")\n}\n"
	diff := []DiffLine{
		{Kind: LineContext, Text: "func F(){"},
		{Kind: LineDel, Text: "  println(\"y\")"}, // doesn't match
		{Kind: LineAdd, Text: "  println(\"z\")"},
		{Kind: LineContext, Text: "}"},
	}
	_, err := ApplyUnified(before, diff)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	me, ok := err.(*MatchError)
	if !ok {
		t.Fatalf("expected MatchError, got %T: %v", err, err)
	}
	// Kind can be "context" when mismatch is detected during the pre-scan.
	if me.Kind != "delete" && me.Kind != "context" {
		t.Fatalf("want Kind=delete or context, got %q", me.Kind)
	}
	if me.Line <= 0 {
		t.Fatalf("want positive line, got %d", me.Line)
	}
	if me.Expected == "" {
		t.Fatalf("expected text missing")
	}
}

