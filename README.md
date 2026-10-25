### gopidgin

Compact agent-facing Go hunks; local expander applies them onto a module, then runs goimports + gofmt.

- Quick start:

```bash
go install github.com/snadrus/gopidgin@latest
gopidgin apply -C example example.gopidgin
```

If you vendor the binary under `cmd/gopidgin`, use:

```bash
go run github.com/snadrus/gopidgin/cmd/gopidgin@latest apply ...
```

### CLI

```bash
gopidgin apply [-C modroot] [-n|--dry-run] [--force] [file|-]
```
- Input: stdin if `-`, else the given file, else `.gopidgin` in module root
- Module root: `-C` or discover via `go env GOMOD`
- Dry-run: prints unified diff, writes nothing
- Exit codes: 0 ok; 2 parse/anchor error; 3 tool/build failure
- Re-applying an already-applied hunk errors unless `--force`

Install the binary to avoid repeated `go run` compilation:

```bash
go install github.com/snadrus/gopidgin@latest
# then:
gopidgin apply ...
```

### Grammar (v0)

Hunk header:

```
@ <relpath> <anchor>
```

- Anchors: `func Name`, `func (*T) Method`, `func (T) Method`, `type Name`, `const Name`, `var Name`, `package`
- New file: `@ <relpath> +` followed by the file body
- Delete file: `@ <relpath> -`

Ops: unified-diff style lines inside the anchored span:
- Leading space = context
- `-` remove
- `+` add

Imports:
- Omit ordinary/default imports; goimports fills them after apply
- Emit import lines ONLY when:
  - renamed: `import sql "database/sql"`
  - blank: `import _ "image/png"`
  - dot: `import . "fmt"`
- Special imports are merged into the real import block before goimports

Aliases (optional, hunk‑local):

```
$name=Identifier
$=Identifier
```

Scope ends at next `@` header. Expand before apply. No support for line‑relative `vN`.

Out of scope for v0:
- Keyword bytecode
- Line‑relative variable refs
- Err‑check macros

### Examples

- Edit a method:

```
@ example/hello/hello.go func (Greeter) Greet
  func (Greeter) Greet() string {
-   return "hi"
+   return "hello"
  }
```

- New file with named and blank imports:

```
@ internal/db/store.go +
package db

import (
  sql "database/sql"
  _ "github.com/mattn/go-sqlite3"
)

type Store struct{ db *sql.DB }
```

- `$` alias:

```
@ pkg/handler.go func New
$r=Request
  h := &handler{$r}
```

### Dev

Run tests:

```bash
go test ./...
```

Try dry‑run on the included fixture:

```bash
go run . apply -n -C example example.gopidgin
```

### Agents and shell usage

- Read only the `.go` files you will edit.
- Produce one pidgin document with all hunks — either write module‑root `.gopidgin` or pipe on stdin.
- Run exactly one apply (no `-n` unless a preview was requested), then optionally `go build`/`go test` once.
- Stop; do not paste the expanded Go back into chat.

Note: Match file indentation (tabs) on `-`/`+` lines when possible; the expander tolerates leading‑whitespace mismatch for context and `-` lines. Go formatters/imports run only for `.go` files; other paths are applied raw.

Write `.gopidgin` then apply (file default):

```bash
printf '%s\n' '@ path.go func Name' '-	old' '+	new' > .gopidgin
gopidgin apply
```

Or heredoc to stdin (explicit `-`):

```bash
gopidgin apply - <<'PIDGIN'
@ path.go func Name
-	old
+	new
PIDGIN
```

Token economics (optimize input vs output separately):
- Output is the expensive side. As a rough score, minimize input + 4×output tokens.
- Be concise in hunks: 0–2 context lines, no ordinary imports, use `$name=Ident` for repeated long names.
- Prefer pidgin for multi‑hunk or many repeated identifiers. For a single tiny edit, regular edit may still be cheaper/faster.
