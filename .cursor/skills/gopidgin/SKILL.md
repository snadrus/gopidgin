---
name: gopidgin
description: Emit compact Go Pidgin hunks and expand with gopidgin apply. Use when editing Go, writing Go patches, or when the user mentions pidgin, gopidgin, or token-cheap Go output.
---

Use this skill to make Go edits with compact Pidgin hunks, then expand with the `gopidgin` CLI in one shot. This minimizes tool churn and tokens.

Strict one‑shot loop (token‑positive):
1) Read only the target `.go` files you will change. Do NOT re‑read this skill once attached; do NOT read `go.mod`; no project‑wide globs.
2) Write/overwrite a single `.gopidgin` at the module root with ALL hunks (one write tool call).
3) Run exactly one shell: `gopidgin apply` (no file arg; defaults to `.gopidgin`) and optionally `go build`/`go test` once.
4) Stop. Do not paste expanded Go back into chat.

Hard bans:
- Do not read or explore anything under `$(go env GOMODCACHE)/github.com/snadrus/gopidgin...`
- Do not read the `github.com/snadrus/gopidgin` repository contents unless the user is developing gopidgin itself
- Do not run `cat -A`, `od`, or hex dumps for indent debugging
- Do not rewrite `.gopidgin` more than once unless apply failed with a clear mismatch error
- Do not use Write/StrReplace on `.go` for the code change (edits must be pidgin + apply)
- Do not re‑read this SKILL file during the loop (load once)
- Do not default to dry‑run

Token economics:
- Optimize input and output separately; output is more expensive. Rough score: input + 4×output tokens.
- Keep hunks minimal (0–2 context lines). Omit ordinary imports. Use `$name=Ident` for repeated long names. Do not reprint unmodified code.
- Prefer pidgin for multi‑hunk changes or when many long identifiers repeat; for a trivial one‑liner, regular edit may be cheaper.

Pidgin v0 you must emit:
- `@ <relpath> <anchor>`: anchors are `func Name`, `func (T) M`, `func (*T) M`, `type Name`, `const Name`, `var Name`, or `package` (never line numbers)
- Unified diff lines inside the span: leading space=context, `-`=remove, `+`=add
- Imports: omit ordinary imports; only emit special ones:
  - renamed: `import sql "database/sql"`
  - blank: `import _ "image/png"`
  - dot: `import . "fmt"`
- Optional hunk‑local aliases for repeated idents:
  - `$name=Identifier` or `$=Identifier`
  - No keyword bytecode; no `vN` line references

Indentation and recovery:
- Copy indentation exactly from the file (gofmt uses tabs for blocks). Prefer pasting the exact current line as the `-` line and only changing `+`.
- If apply fails, re‑read the file and fix the hunk indent; expander tolerates leading‑whitespace mismatch on context/`-` lines, but exact matches are more reliable.
- Only edit `.gopidgin` (or stdin), never `.go` directly.

How to expand:
- Cursor/agent canonical: write `.gopidgin` then run apply (no file arg):

```bash
printf '%s\n' '@ path.go func Name' '-	old' '+	new' > .gopidgin
gopidgin apply
```

- CLI / optional: heredoc to stdin (explicit `-`):

```bash
gopidgin apply - <<'PIDGIN'
@ path.go func Name
-	old
+	new
PIDGIN
```

Grammar and more examples: see the repo README.

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

