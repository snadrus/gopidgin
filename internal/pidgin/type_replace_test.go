package pidgin_test

import (
	"path/filepath"
	"testing"
	"os"
	"strings"

	"github.com/snadrus/gopidgin/internal/anchor"
	pid "github.com/snadrus/gopidgin/internal/pidgin"
)

func TestWholeSpanReplace_TypeDeclaration(t *testing.T) {
	src := `package p

type Hunk struct {
	A string
}
`
	tmp := t.TempDir()
	p := filepath.Join(tmp, "x.go")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// Resolve type span
	span, err := anchor.Resolve(p, src, pid.AnchorSpec{Kind: pid.AnchorType, Name: "Hunk"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// Prepare +only replacement
	diff := []pid.DiffLine{
		{Kind: pid.LineAdd, Text: "type Hunk struct {"},
		{Kind: pid.LineAdd, Text: "  A int"},
		{Kind: pid.LineAdd, Text: "}"},
	}
	newSpan, err := pid.ApplyUnified(span.Text, diff)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	final := src[:span.StartOffset] + newSpan + src[span.EndOffset:]
	if !strings.Contains(final, "type Hunk") || !strings.Contains(final, "A int") {
		t.Fatalf("missing replacements:\n%s", final)
	}
}

func TestSurgicalEdit_TypeField(t *testing.T) {
	src := `package p

type Hunk struct {
	A string
}
`
	tmp := t.TempDir()
	p := filepath.Join(tmp, "x.go")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	span, err := anchor.Resolve(p, src, pid.AnchorSpec{Kind: pid.AnchorType, Name: "Hunk"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	diff := []pid.DiffLine{
		{Kind: pid.LineContext, Text: "type Hunk struct {"},
		{Kind: pid.LineDel, Text: "	A string"},
		{Kind: pid.LineAdd, Text: "	A int"},
		{Kind: pid.LineContext, Text: "}"},
	}
	newSpan, err := pid.ApplyUnified(span.Text, diff)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	final := src[:span.StartOffset] + newSpan + src[span.EndOffset:]
	if !strings.Contains(final, "A int") || strings.Contains(final, "A string") {
		t.Fatalf("surgical edit failed:\n%s", final)
	}
}
