package anchor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/snadrus/gopidgin/internal/pidgin"
)

type Span struct {
	StartOffset int
	EndOffset   int
	Text        string
}

// Resolve finds the anchored span in source according to spec.
func Resolve(absPath, src string, spec pidgin.AnchorSpec) (Span, error) {
	if spec.Kind == pidgin.AnchorPackage {
		return Span{StartOffset: 0, EndOffset: len(src), Text: src}, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, src, parser.ParseComments)
	if err != nil {
		return Span{}, fmt.Errorf("parse file: %w", err)
	}
	start, end, err := locate(fset, file, spec)
	if err != nil {
		return Span{}, err
	}
	offsetStart := fset.PositionFor(start, false).Offset
	offsetEnd := fset.PositionFor(end, false).Offset
	if offsetStart < 0 || offsetEnd < offsetStart || offsetEnd > len(src) {
		return Span{}, fmt.Errorf("invalid span resolved")
	}
	return Span{
		StartOffset: offsetStart,
		EndOffset:   offsetEnd,
		Text:        src[offsetStart:offsetEnd],
	}, nil
}

func locate(fset *token.FileSet, f *ast.File, spec pidgin.AnchorSpec) (token.Pos, token.Pos, error) {
	switch spec.Kind {
	case pidgin.AnchorFunc:
		fn := findFunc(f, spec.Name, "")
		if fn == nil {
			return 0, 0, notFound(f, spec)
		}
		return fn.Pos(), fn.End(), nil
	case pidgin.AnchorMethod:
		fn := findFunc(f, spec.Name, spec.Recv)
		if fn == nil {
			return 0, 0, notFound(f, spec)
		}
		return fn.Pos(), fn.End(), nil
	case pidgin.AnchorType:
		ts := findType(f, spec.Name)
		if ts == nil {
			return 0, 0, notFound(f, spec)
		}
		return ts.Pos(), ts.End(), nil
	case pidgin.AnchorConst:
		vs := findValueSpec(f, token.CONST, spec.Name)
		if vs == nil {
			return 0, 0, notFound(f, spec)
		}
		return vs.Pos(), vs.End(), nil
	case pidgin.AnchorVar:
		vs := findValueSpec(f, token.VAR, spec.Name)
		if vs == nil {
			return 0, 0, notFound(f, spec)
		}
		return vs.Pos(), vs.End(), nil
	default:
		return 0, 0, fmt.Errorf("unsupported anchor kind")
	}
}

func findFunc(f *ast.File, name string, recv string) *ast.FuncDecl {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Name == nil || fd.Name.Name != name {
			continue
		}
		if recv == "" && fd.Recv == nil {
			return fd
		}
		if recv != "" && fd.Recv != nil {
			if receiverMatches(fd.Recv, recv) {
				return fd
			}
		}
	}
	return nil
}

func receiverMatches(fieldList *ast.FieldList, recv string) bool {
	// recv is like "T" or "*T" (or with param name? Spec says no name)
	target := strings.TrimSpace(recv)
	// Consider only first receiver
	if fieldList.NumFields() == 0 {
		return false
	}
	ft := fieldList.List[0].Type
	switch t := ft.(type) {
	case *ast.StarExpr:
		ident, ok := t.X.(*ast.Ident)
		if !ok {
			return false
		}
		return target == "*"+ident.Name
	case *ast.Ident:
		return target == t.Name
	default:
		return false
	}
}

func findType(f *ast.File, name string) *ast.TypeSpec {
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.TYPE {
			continue
		}
		for _, spec := range g.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if ts.Name != nil && ts.Name.Name == name {
				return ts
			}
		}
	}
	return nil
}

func findValueSpec(f *ast.File, tok token.Token, name string) *ast.ValueSpec {
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != tok {
			continue
		}
		for _, spec := range g.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, id := range vs.Names {
				if id.Name == name {
					return vs
				}
			}
		}
	}
	return nil
}

func notFound(f *ast.File, spec pidgin.AnchorSpec) error {
	// Produce some candidates to help
	var cands []string
	switch spec.Kind {
	case pidgin.AnchorFunc, pidgin.AnchorMethod:
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Name != nil {
				name := fd.Name.Name
				if fd.Recv != nil && fd.Recv.NumFields() > 0 {
					recv := fd.Recv.List[0].Type
					var rstr string
					switch t := recv.(type) {
					case *ast.StarExpr:
						if id, ok := t.X.(*ast.Ident); ok {
							rstr = "*" + id.Name
						}
					case *ast.Ident:
						rstr = t.Name
					}
					cands = append(cands, fmt.Sprintf("func (%s) %s", rstr, name))
				} else {
					cands = append(cands, fmt.Sprintf("func %s", name))
				}
			}
		}
	case pidgin.AnchorType:
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, s := range g.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && ts.Name != nil {
						cands = append(cands, "type "+ts.Name.Name)
					}
				}
			}
		}
	case pidgin.AnchorConst, pidgin.AnchorVar:
		kind := "const"
		if spec.Kind == pidgin.AnchorVar {
			kind = "var"
		}
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok {
				if (kind == "const" && g.Tok != token.CONST) || (kind == "var" && g.Tok != token.VAR) {
					continue
				}
				for _, s := range g.Specs {
					if vs, ok := s.(*ast.ValueSpec); ok {
						for _, n := range vs.Names {
							cands = append(cands, kind+" "+n.Name)
						}
					}
				}
			}
		}
	}
	p := filepath.Base(f.Name.Name) // best-effort package name
	return fmt.Errorf("anchor %q not found in %s; candidates: %s", spec.String(), p, strings.Join(cands, ", "))
}

