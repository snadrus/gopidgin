package work

import (
	"strings"
	"testing"

	"github.com/snadrus/gopidgin/internal/pidgin"
)

func TestMergeSpecialImports(t *testing.T) {
	in := "package p\n\nfunc F(){}\n"
	out, err := MergeSpecialImports("x.go", in, []pidgin.ImportSpec{
		{Name: "sql", Path: "database/sql"},
		{Name: "_", Path: "image/png"},
		{Name: ".", Path: "fmt"},
	}, nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if !strings.Contains(out, `import (`) {
		t.Fatalf("want import block, got:\n%s", out)
	}
	if !strings.Contains(out, `sql "database/sql"`) ||
		!strings.Contains(out, `_ "image/png"`) ||
		!strings.Contains(out, `. "fmt"`) {
		t.Fatalf("imports missing:\n%s", out)
	}
}

func TestGoImportsThenFmt(t *testing.T) {
	src := "package p\n\nfunc F(){_ = fmt.Sprintln(1)}\n"
	b, err := GoImportsThenFmt("x.go", []byte(src))
	if err != nil {
		t.Fatalf("goimports: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, `"fmt"`) {
		t.Fatalf("goimports did not add fmt import:\n%s", out)
	}
	if !strings.Contains(out, "func F()") {
		t.Fatalf("gofmt did not format function")
	}
}

