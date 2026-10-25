### gopidgin

Compact agent-facing Go hunks; local expander applies them onto a module, then runs goimports + gofmt.

- One‑liner:

```bash
go run github.com/snadrus/gopidgin@latest apply -n -C example example.gopidgin
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

### Cursor agents

- **Install the skill into a project**:

```bash
mkdir -p .cursor/skills/gopidgin && \
curl -fsSL https://raw.githubusercontent.com/snadrus/gopidgin/main/.cursor/skills/gopidgin/SKILL.md \
  -o .cursor/skills/gopidgin/SKILL.md
```

- **Install the skill globally** (for all projects):

```bash
mkdir -p ~/.cursor/skills/gopidgin && \
curl -fsSL https://raw.githubusercontent.com/snadrus/gopidgin/main/.cursor/skills/gopidgin/SKILL.md \
  -o ~/.cursor/skills/gopidgin/SKILL.md
```

- **Use it**:
  - In an Agent chat, type `/gopidgin` to switch into Pidgin mode (Option/Alt+Enter to keep as Custom Mode).
  - Emit only Go Pidgin hunks for edits; then expand locally:

```bash
go run github.com/snadrus/gopidgin@latest apply [-n] [-C modroot] [file|-]
```

- **If the skill isn't installed**:
  - Ask the agent to output Go Pidgin hunks and run the CLI manually as above.

Note: Match file indentation (tabs) on `-`/`+` lines when possible; the expander also tolerates leading‑whitespace mismatch for context and `-` lines.
