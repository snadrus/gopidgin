---
name: gopidgin
description: Emit compact Go Pidgin hunks and expand locally with `gopidgin apply` to avoid streaming full gofmt/goimports noise.
---

Audience: any coding agent or human using a shell and making file edits.

Invoke:
- Default invocation is always via `go run` to avoid PATH assumptions:

```bash
go run github.com/snadrus/gopidgin@latest apply [flags]
```

One‑shot loop (tool‑agnostic):
1) Read only the Go source files you will change (avoid re‑reading this skill mid‑loop; avoid reading go.mod or globbing unrelated paths).
2) Produce one Pidgin document with all hunks — either write it as module‑root `.gopidgin` or pipe it on stdin.
3) Run exactly one `go run github.com/snadrus/gopidgin@latest apply` (file default uses `.gopidgin`; use `apply -` for stdin). Optionally run one `go build` or `go test`.
4) Stop. Do not paste the expanded Go back into the chat unless the user asks for it.

No dry‑run unless the user explicitly requests a preview.

Recovery protocol (avoid token waste):
- If `apply` reports a mismatch, re‑read the target source file (not the gopidgin sources) and fix the pidgin document once using the expected/got hint, then run `apply` again for real.
- Do not use dry‑run for recovery, do not probe with heredocs, and do not call `-h`.
- Prefer one successful apply; a second apply is only after that single correction.

General bans (keep tokens low and avoid dead ends):
- Do not read or explore anything under `$(go env GOMODCACHE)/github.com/snadrus/gopidgin...`
- Do not read the `github.com/snadrus/gopidgin` repository contents unless you are developing gopidgin itself
- Do not hex‑dump or `cat -A` for indentation; instead, copy the exact line from the file
- Do not re‑write `.gopidgin` more than once unless `apply` returns a clear mismatch error
- Do not edit `.go` files directly when using this skill; emit pidgin and apply
- Do not re‑read this skill during the loop

Token tip:
- Optimize input and output separately; output is more expensive. Rough score: input + 4×output tokens.
- Keep hunks minimal (0–2 context lines). Omit ordinary imports. Use `$name=Ident` for repeated long names. Do not reprint unmodified code.
- Prefer pidgin for multi‑hunk changes or when many long identifiers repeat. For a trivial one‑liner, a direct edit may still be cheaper.

Pidgin v0 to emit:
- `@ <relpath> <anchor>`: anchors are `func Name`, `func (T) M`, `func (*T) M`, `type Name`, `const Name`, `var Name`, or `package` (never line numbers)
- Unified diff lines inside the span: leading space = context, `-` = remove, `+` = add
- Imports: omit ordinary imports; only emit special ones:
  - renamed: `import sql "database/sql"`
  - blank: `import _ "image/png"`
  - dot: `import . "fmt"`
- Optional hunk‑local aliases for repeated identifiers:
  - `$name=Identifier` or `$=Identifier`
  - No keyword bytecode; no `vN` line references

Indentation and recovery:
- Copy indentation exactly from the file (gofmt uses tabs for blocks). Prefer pasting the exact current line as the `-` line and only changing `+`.
- If apply fails, re‑read the file and fix the hunk indent; the expander tolerates leading‑whitespace mismatch on context/`-` lines, but exact matches are most reliable.
- Go formatters/imports run only for `.go` files; other paths are applied raw.

How to expand (either approach is fine — pick one and succeed on the first apply):
- Write `.gopidgin` then run apply (file default):

```bash
printf '%s\n' '@ path.go func Name' '-	old' '+	new' > .gopidgin
go run github.com/snadrus/gopidgin@latest apply
```

- Or pipe to stdin explicitly:

```bash
go run github.com/snadrus/gopidgin@latest apply - <<'PIDGIN'
@ path.go func Name
-	old
+	new
PIDGIN
```

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

