package anchor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snadrus/gopidgin/internal/pidgin"
)

func TestResolveFuncAndMethod(t *testing.T) {
	path := filepath.Join("..", "..", "example", "hello", "hello.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Method anchor
	span, err := Resolve(path, string(data), pidgin.AnchorSpec{
		Kind: pidgin.AnchorMethod, Name: "Greet", Recv: "Greeter",
	})
	if err != nil {
		t.Fatalf("resolve method: %v", err)
	}
	if span.StartOffset >= span.EndOffset {
		t.Fatalf("bad span")
	}
	// Package anchor
	span2, err := Resolve(path, string(data), pidgin.AnchorSpec{Kind: pidgin.AnchorPackage})
	if err != nil {
		t.Fatalf("resolve package: %v", err)
	}
	if span2.Text == "" || span2.StartOffset != 0 {
		t.Fatalf("bad package span")
	}
}

