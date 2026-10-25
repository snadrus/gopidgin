---
name: gopidgin
description: Emit compact Go Pidgin hunks and expand with gopidgin apply. Use when editing Go, writing Go patches, or when the user mentions pidgin, gopidgin, or token-cheap Go output.
---

Use this skill whenever you need to change Go code. Prefer emitting compact Pidgin hunks over full-file edits. Expand locally with the `gopidgin` CLI.

What to emit (Pidgin v0):
- Only Pidgin hunks, never full `.go` files for edits. New files use a create hunk.
- Anchor each hunk with `@ <relpath> <anchor>` where anchor is one of:
  - `func Name`, `func (T) Method`, `func (*T) Method`, `type Name`, `const Name`, `var Name`, or `package`
  - Never use line numbers
- Inside hunks, use unified diff lines:
  - Leading space = context, `-` = remove, `+` = add
- Imports:
  - Omit ordinary/default imports entirely; goimports will add them
  - Emit ONLY special imports:
    - renamed: `import sql "database/sql"`
    - blank: `import _ "image/png"`
    - dot: `import . "fmt"`
- Optional hunk‑local aliases for brevity:
  - `$name=Identifier` or `$=Identifier`
  - No keyword bytecode; no `vN` line references

Indentation rules and recovery:
- Before writing a hunk, Read the target `.go` file and copy indentation exactly (gofmt uses tabs for blocks). Do not guess spaces.
- Prefer including the exact current line as the `-` line (byte‑for‑byte from the file) and only change what must change on `+`.
- If apply fails: re‑read the file and fix the hunk indent. Do not explore gopidgin’s module cache unless apply still fails twice.
- Still: only edit `.gopidgin` (or stdin), never Write/ApplyPatch `.go` files for edits; expand with the CLI below.

How to expand:
- Write the hunks to a temporary file (e.g., `.gopidgin`) or pipe via stdin.
- Run the expander at the module root (or pass `-C`):

```bash
go run github.com/snadrus/gopidgin@latest apply [-n] [-C modroot] [file|-]
```

Guidance:
- Prefer `-n/--dry-run` first to preview a unified diff; then apply for real.
- Do not paste the expanded Go back into chat unless the user explicitly asks.
- If an anchor fails to resolve, fix the anchor (never switch to line numbers).
- Grammar and examples: see the repo README.

Create hunk example:

```
@ internal/db/store.go +
package db

import (
  sql "database/sql"
  _ "github.com/mattn/go-sqlite3"
)

type Store struct{ db *sql.DB }
```

Edit hunk example:

```
@ pkg/hello/hello.go func (Greeter) Greet
  func (Greeter) Greet() string {
-   return "hi"
+   return "hello"
  }
```

