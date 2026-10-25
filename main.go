package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/snadrus/gopidgin/internal/anchor"
	"github.com/snadrus/gopidgin/internal/pidgin"
	"github.com/snadrus/gopidgin/internal/work"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "apply":
		exitCode := runApply(os.Args[2:])
		os.Exit(exitCode)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("gopidgin — expand pidgin hunks onto a Go module")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gopidgin apply [-C modroot] [-n|--dry-run] [--force] [file|-]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -C string       Module root (defaults to discovery via `go env GOMOD`)")
	fmt.Println("  -n, --dry-run   Show unified diff, do not write")
	fmt.Println("  --force         Do not error if hunks are already applied")
	fmt.Println()
	fmt.Println("Input: stdin if '-' or file omitted (defaults to .gopidgin in module root)")
}

func runApply(args []string) int {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	modRoot := fs.String("C", "", "module root")
	dryRun := fs.Bool("n", false, "dry-run")
	dryRunLong := fs.Bool("dry-run", false, "dry-run")
	force := fs.Bool("force", false, "force re-apply")
	fs.SetOutput(io.Discard) // keep stdout clean; errors printed below
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	isDry := *dryRun || *dryRunLong

	var root string
	if *modRoot != "" {
		root = *modRoot
	} else {
		var err error
		root, err = discoverModuleRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, "module root discovery failed:", err)
			return 3
		}
	}
	root, _ = filepath.Abs(root)

	var inPath string
	var rdr io.Reader
	rest := fs.Args()
	if len(rest) == 0 {
		inPath = filepath.Join(root, ".gopidgin")
		f, err := os.Open(inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "input file not found: %s\n", inPath)
			return 2
		}
		defer f.Close()
		rdr = f
	} else if rest[0] == "-" {
		rdr = bufio.NewReader(os.Stdin)
		inPath = "stdin"
	} else {
		inPath = rest[0]
		f, err := os.Open(inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open input: %v\n", err)
			return 2
		}
		defer f.Close()
		rdr = f
	}

	src, err := io.ReadAll(rdr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read error:", err)
		return 3
	}

	hunks, perr := pidgin.Parse(string(src))
	if perr != nil {
		fmt.Fprintln(os.Stderr, "parse error:", perr)
		return 2
	}

	// Apply hunks
	touched := map[string]struct{}{}
	diffs := []string{}

	for _, h := range hunks {
		abs := filepath.Join(root, filepath.FromSlash(h.RelPath))
		switch h.Action {
		case pidgin.ActionDelete:
			// capture diff if file exists
			if data, err := os.ReadFile(abs); err == nil {
				if isDry {
					diff, _ := unified(abs, string(data), "")
					diffs = append(diffs, diff)
				} else {
					if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
						fmt.Fprintf(os.Stderr, "delete failed for %s: %v\n", h.RelPath, err)
						return 3
					}
				}
			} else if !*force && !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", h.RelPath, err)
				return 3
			}
			continue
		case pidgin.ActionCreate:
			if _, err := os.Stat(abs); err == nil && !*force {
				fmt.Fprintf(os.Stderr, "file already exists: %s (use --force)\n", h.RelPath)
				return 2
			}
			dir := filepath.Dir(abs)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "mkdir failed for %s: %v\n", dir, err)
				return 3
			}
			current := ""
			newContent := strings.Join(h.RawBody, "\n")
			if isDry {
				diff, _ := unified(abs, current, newContent)
				diffs = append(diffs, diff)
			} else {
				if err := os.WriteFile(abs, []byte(newContent), 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "write failed for %s: %v\n", h.RelPath, err)
					return 3
				}
				touched[abs] = struct{}{}
			}
			continue
		case pidgin.ActionModify:
			// Proceed
		default:
			fmt.Fprintf(os.Stderr, "unsupported action for %s\n", h.RelPath)
			return 2
		}

		// Read file
		origBytes, err := os.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read file: %s: %v\n", h.RelPath, err)
			return 3
		}
		orig := string(origBytes)

		// Resolve anchor
		span, aerr := anchor.Resolve(abs, orig, h.Anchor)
		if aerr != nil {
			fmt.Fprintf(os.Stderr, "anchor error for %s: %v\n", h.RelPath, aerr)
			return 2
		}

		// Apply unified hunk to span
		newSpan, apErr := pidgin.ApplyUnified(span.Text, h.Diff)
		if apErr != nil {
			// Surface actionable mismatch with file line numbers when possible
			if me, ok := apErr.(*pidgin.MatchError); ok {
				fileLine := 0
				if me.Line > 0 {
					prefix := orig[:span.StartOffset]
					fileLine = 1 + bytes.Count([]byte(prefix), []byte("\n")) + me.Line - 1
				}
				if fileLine > 0 {
					fmt.Fprintf(os.Stderr, "%s:%d: %s\n", h.RelPath, fileLine, me.Error())
				} else {
					fmt.Fprintf(os.Stderr, "%s: %s\n", h.RelPath, me.Error())
				}
			} else {
				fmt.Fprintf(os.Stderr, "apply error for %s: %v\n", h.RelPath, apErr)
			}
			return 2
		}

		// Merge into file
		var buf bytes.Buffer
		buf.WriteString(orig[:span.StartOffset])
		buf.WriteString(newSpan)
		buf.WriteString(orig[span.EndOffset:])
		applied := buf.String()

		if applied == orig && !*force {
			fmt.Fprintf(os.Stderr, "already applied: %s (%s)\n", h.RelPath, h.Anchor.String())
			return 2
		}

		// Handle special imports on the applied content
		applied, ierr := work.MergeSpecialImports(abs, applied, h.SpecialImportsAdd, h.SpecialImportsDel)
		if ierr != nil {
			fmt.Fprintf(os.Stderr, "import merge error for %s: %v\n", h.RelPath, ierr)
			return 3
		}

		// goimports then gofmt
		formatted, ferr := work.GoImportsThenFmt(abs, []byte(applied))
		if ferr != nil {
			fmt.Fprintf(os.Stderr, "format error for %s: %v\n", h.RelPath, ferr)
			return 3
		}
		final := string(formatted)

		if isDry {
			diff, _ := unified(abs, orig, final)
			diffs = append(diffs, diff)
		} else {
			if err := os.WriteFile(abs, []byte(final), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "write failed for %s: %v\n", h.RelPath, err)
				return 3
			}
			touched[abs] = struct{}{}
		}
	}

	// For created files, run goimports+gofmt and merge imports if needed
	if !isDry {
		for abs := range touched {
			b, _ := os.ReadFile(abs)
			out, ferr := work.GoImportsThenFmt(abs, b)
			if ferr != nil {
				fmt.Fprintf(os.Stderr, "format error for %s: %v\n", abs, ferr)
				return 3
			}
			if !bytes.Equal(out, b) {
				if err := os.WriteFile(abs, out, 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "write failed for %s: %v\n", abs, err)
					return 3
				}
			}
		}
	}

	if isDry {
		for _, d := range diffs {
			if strings.TrimSpace(d) == "" {
				continue
			}
			fmt.Println(d)
		}
	}

	return 0
}

func unified(path, before, after string) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(before),
		B:        difflib.SplitLines(after),
		FromFile: "a/" + filepath.ToSlash(path),
		ToFile:   "b/" + filepath.ToSlash(path),
		Context:  3,
	}
	var buf strings.Builder
	if err := difflib.WriteUnifiedDiff(&buf, diff); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func discoverModuleRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go env GOMOD failed: %v", err)
	}
	out := strings.TrimSpace(cmd.Stdout.(*bytes.Buffer).String())
	if out == "" || out == os.DevNull {
		// Search upwards for go.mod
		dir, _ := os.Getwd()
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		return "", errors.New("no go.mod found")
	}
	return filepath.Dir(out), nil
}

