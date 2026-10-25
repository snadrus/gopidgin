package pidgin

import (
	"fmt"
	"strings"
)

type Action int

const (
	ActionModify Action = iota
	ActionCreate
	ActionDelete
)

type AnchorKind int

const (
	AnchorPackage AnchorKind = iota
	AnchorFunc
	AnchorMethod
	AnchorType
	AnchorConst
	AnchorVar
)

type AnchorSpec struct {
	Kind         AnchorKind
	Name         string // for func/type/const/var
	Recv         string // for method: T or *T (exactly as in anchor)
	PrettyCached string // memoized string
}

func (a AnchorSpec) String() string {
	if a.PrettyCached != "" {
		return a.PrettyCached
	}
	switch a.Kind {
	case AnchorPackage:
		return "package"
	case AnchorFunc:
		return fmt.Sprintf("func %s", a.Name)
	case AnchorType:
		return fmt.Sprintf("type %s", a.Name)
	case AnchorConst:
		return fmt.Sprintf("const %s", a.Name)
	case AnchorVar:
		return fmt.Sprintf("var %s", a.Name)
	case AnchorMethod:
		return fmt.Sprintf("func (%s) %s", a.Recv, a.Name)
	default:
		return "?"
	}
}

type DiffLineKind int

const (
	LineContext DiffLineKind = iota
	LineAdd
	LineDel
)

type DiffLine struct {
	Kind DiffLineKind
	Text string
}

type ImportSpec struct {
	Name string // alias, "_" or "."
	Path string // quoted or unquoted; we'll normalize
}

// Hunk represents one pidgin hunk.
type Hunk struct {
	RelPath           string
	Action            Action
	Anchor            AnchorSpec
	Aliases           map[string]string
	Diff              []DiffLine
	RawBody           []string // for create
	SpecialImportsAdd []ImportSpec
	SpecialImportsDel []ImportSpec
}

func (h *Hunk) expandAliasesInPlace() {
	if len(h.Aliases) == 0 {
		return
	}
	repls := make([][2]string, 0, len(h.Aliases))
	var defaultAlias string
	for k, v := range h.Aliases {
		if k == "" {
			defaultAlias = v
			continue
		}
		repls = append(repls, [2]string{"$" + k, v})
	}
	replaceAll := func(s string) string {
		for _, p := range repls {
			s = strings.ReplaceAll(s, p[0], p[1])
		}
		if defaultAlias != "" {
			// Replace standalone $ when not followed by ident char
			s = strings.ReplaceAll(s, "$", defaultAlias)
		}
		return s
	}
	if h.Action == ActionCreate {
		for i := range h.RawBody {
			h.RawBody[i] = replaceAll(h.RawBody[i])
		}
		return
	}
	for i := range h.Diff {
		h.Diff[i].Text = replaceAll(h.Diff[i].Text)
	}
}

