package work

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"golang.org/x/tools/imports"

	"github.com/snadrus/gopidgin/internal/pidgin"
)

// MergeSpecialImports merges special imports (renamed, blank, dot) into the file content.
func MergeSpecialImports(filename string, src string, add []pidgin.ImportSpec, del []pidgin.ImportSpec) (string, error) {
	if len(add) == 0 && len(del) == 0 {
		return src, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}

	// Normalize path quotes and build sets
	norm := func(s pidgin.ImportSpec) pidgin.ImportSpec {
		path := s.Path
		if !strings.HasPrefix(path, `"`) {
			path = `"` + path + `"`
		}
		if !strings.HasSuffix(path, `"`) {
			path = path + `"`
		}
		return pidgin.ImportSpec{Name: s.Name, Path: path}
	}
	wantAdd := map[string]pidgin.ImportSpec{}
	for _, s := range add {
		s = norm(s)
		key := s.Name + " " + s.Path
		wantAdd[key] = s
	}
	wantDel := map[string]pidgin.ImportSpec{}
	for _, s := range del {
		s = norm(s)
		key := s.Name + " " + s.Path
		wantDel[key] = s
	}

	// Find or create import decl
	var impDecl *ast.GenDecl
	for _, d := range file.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			impDecl = g
			break
		}
	}
	if impDecl == nil {
		impDecl = &ast.GenDecl{Tok: token.IMPORT, Lparen: 1} // force block form
		// Insert at top (after package and comments)
		file.Decls = append([]ast.Decl{impDecl}, file.Decls...)
	}
	if impDecl.Lparen == token.NoPos && (len(add) > 1 || len(impDecl.Specs) > 0) {
		// convert to block form
		impDecl.Lparen = 1
	}

	// Build current set for quick lookup and apply deletes
	cur := map[string]int{} // key -> index
	for i, sp := range impDecl.Specs {
		is, ok := sp.(*ast.ImportSpec)
		if !ok {
			continue
		}
		name := ""
		if is.Name != nil {
			name = is.Name.Name
		}
		key := name + " " + is.Path.Value
		cur[key] = i
	}
	// Deletes
	if len(wantDel) > 0 {
		newSpecs := make([]ast.Spec, 0, len(impDecl.Specs))
		for i, sp := range impDecl.Specs {
			is, ok := sp.(*ast.ImportSpec)
			if !ok {
				newSpecs = append(newSpecs, sp)
				continue
			}
			n := ""
			if is.Name != nil {
				n = is.Name.Name
			}
			key := n + " " + is.Path.Value
			if _, drop := wantDel[key]; drop {
				continue // skip
			}
			_ = i
			newSpecs = append(newSpecs, sp)
		}
		impDecl.Specs = newSpecs
	}
	// Adds
	for _, s := range wantAdd {
		key := s.Name + " " + s.Path
		if _, exists := cur[key]; exists {
			continue
		}
		spec := &ast.ImportSpec{
			Path: &ast.BasicLit{Kind: token.STRING, Value: s.Path},
		}
		if s.Name != "" && s.Name != "_" && s.Name != "." {
			spec.Name = &ast.Ident{Name: s.Name}
		} else if s.Name == "_" || s.Name == "." {
			spec.Name = &ast.Ident{Name: s.Name}
		}
		impDecl.Specs = append(impDecl.Specs, spec)
		if len(impDecl.Specs) > 1 {
			impDecl.Lparen = 1
		}
	}

	// Print back to source
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return "", fmt.Errorf("format: %w", err)
	}
	return buf.String(), nil
}

func GoImportsThenFmt(filename string, src []byte) ([]byte, error) {
	cfg := &imports.Options{
		AllErrors:  true,
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false,
	}
	processed, err := imports.Process(filename, src, cfg)
	if err != nil {
		// Return code 3 category (tool/build failure)
		return nil, fmt.Errorf("goimports: %w", err)
	}
	formatted, err := format.Source(processed)
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w", err)
	}
	return formatted, nil
}

