package pidgin

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

var (
	headerRe       = regexp.MustCompile(`^@\s+(\S+)\s+(.+)\s*$`)
	aliasRe        = regexp.MustCompile(`^\$(\w*)=(\S+)\s*$`)
	importLineRe   = regexp.MustCompile(`^\s*import\s+(.+)$`)
	renamedImport  = regexp.MustCompile(`^([\.\_A-Za-z]\w*)\s+"([^"]+)"$`)
	plainQuotedImp = regexp.MustCompile(`^"([^"]+)"$`)
)

// Parse parses a pidgin file into hunks.
func Parse(input string) ([]Hunk, error) {
	s := bufio.NewScanner(strings.NewReader(input))
	s.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	var hunks []Hunk
	var cur *Hunk
	lineNo := 0
	for s.Scan() {
		lineNo++
		raw := s.Text()
		if strings.TrimSpace(raw) == "" && cur == nil {
			continue
		}
		if strings.HasPrefix(raw, "@ ") {
			// Start new hunk
			m := headerRe.FindStringSubmatch(raw)
			if m == nil {
				return nil, fmt.Errorf("line %d: invalid header: %q", lineNo, raw)
			}
			if cur != nil {
				cur.expandAliasesInPlace()
				hunks = append(hunks, *cur)
			}
			rel := m[1]
			anchorText := strings.TrimSpace(m[2])
			h, err := parseHeader(rel, anchorText)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			cur = &h
			continue
		}
		if cur == nil {
			return nil, fmt.Errorf("line %d: content before first header", lineNo)
		}
		// Alias directive?
		if strings.HasPrefix(raw, "$") {
			m := aliasRe.FindStringSubmatch(raw)
			if m == nil {
				return nil, fmt.Errorf("line %d: invalid alias: %q", lineNo, raw)
			}
			if cur.Aliases == nil {
				cur.Aliases = map[string]string{}
			}
			name := m[1] // may be empty => default
			cur.Aliases[name] = m[2]
			continue
		}
		if cur.Action == ActionCreate {
			// For create: treat body as raw lines. If it has unified markers, strip leading '+' for convenience.
			body := raw
			if strings.HasPrefix(body, "+") {
				body = strings.TrimPrefix(body, "+")
			}
			cur.RawBody = append(cur.RawBody, body)
			continue
		}
		// Unified diff line for modify/delete within anchor
		if len(raw) == 0 {
			continue
		}
		prefix := raw[0]
		text := raw
		switch prefix {
		case ' ', '+', '-':
			text = strings.TrimPrefix(text, string(prefix))
		default:
			// Treat as context to be lenient
			prefix = ' '
		}
		// Detect special import add/remove
		if (prefix == '+' || prefix == '-') && importLineRe.MatchString(text) {
			spec := strings.TrimSpace(importLineRe.FindStringSubmatch(text)[1])
			if ms := renamedImport.FindStringSubmatch(spec); ms != nil {
				imp := ImportSpec{Name: ms[1], Path: ms[2]}
				if prefix == '+' {
					cur.SpecialImportsAdd = append(cur.SpecialImportsAdd, imp)
				} else {
					cur.SpecialImportsDel = append(cur.SpecialImportsDel, imp)
				}
				// Do not include as content line; import will be merged separately
				continue
			}
			// plain import "x" is not allowed in pidgin body — ordinary imports omitted
			if plainQuotedImp.MatchString(spec) {
				return nil, fmt.Errorf("line %d: default imports must be omitted; use goimports", lineNo)
			}
		}

		switch prefix {
		case ' ':
			cur.Diff = append(cur.Diff, DiffLine{Kind: LineContext, Text: text})
		case '+':
			cur.Diff = append(cur.Diff, DiffLine{Kind: LineAdd, Text: text})
		case '-':
			cur.Diff = append(cur.Diff, DiffLine{Kind: LineDel, Text: text})
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if cur != nil {
		cur.expandAliasesInPlace()
		hunks = append(hunks, *cur)
	}
	return hunks, nil
}

func parseHeader(rel, anchorText string) (Hunk, error) {
	anchorText = strings.TrimSpace(anchorText)
	if anchorText == "+" {
		return Hunk{RelPath: rel, Action: ActionCreate}, nil
	}
	if anchorText == "-" {
		return Hunk{RelPath: rel, Action: ActionDelete}, nil
	}
	// Anchored modify
	spec, err := parseAnchorSpec(anchorText)
	if err != nil {
		return Hunk{}, err
	}
	return Hunk{
		RelPath: rel,
		Action:  ActionModify,
		Anchor:  spec,
	}, nil
}

func parseAnchorSpec(s string) (AnchorSpec, error) {
	switch {
	case s == "package":
		return AnchorSpec{Kind: AnchorPackage, PrettyCached: "package"}, nil
	case strings.HasPrefix(s, "func "):
		rest := strings.TrimSpace(strings.TrimPrefix(s, "func "))
		// method?
		if strings.HasPrefix(rest, "(") {
			// func (T) Name or func (*T) Name
			rparen := strings.Index(rest, ")")
			if rparen < 0 {
				return AnchorSpec{}, fmt.Errorf("invalid method anchor: %q", s)
			}
			recv := strings.TrimSpace(rest[1:rparen])
			after := strings.TrimSpace(rest[rparen+1:])
			if !strings.HasPrefix(after, " ") && after != "" {
				// ensure a space is seen as separator
				after = strings.TrimSpace(after)
			}
			name := strings.TrimSpace(after)
			if name == "" {
				return AnchorSpec{}, fmt.Errorf("invalid method anchor: %q", s)
			}
			return AnchorSpec{Kind: AnchorMethod, Name: name, Recv: recv, PrettyCached: s}, nil
		}
		// plain func
		name := strings.TrimSpace(rest)
		if name == "" {
			return AnchorSpec{}, fmt.Errorf("invalid func anchor: %q", s)
		}
		return AnchorSpec{Kind: AnchorFunc, Name: name, PrettyCached: s}, nil
	case strings.HasPrefix(s, "type "):
		name := strings.TrimSpace(strings.TrimPrefix(s, "type "))
		if name == "" {
			return AnchorSpec{}, fmt.Errorf("invalid type anchor: %q", s)
		}
		return AnchorSpec{Kind: AnchorType, Name: name, PrettyCached: s}, nil
	case strings.HasPrefix(s, "const "):
		name := strings.TrimSpace(strings.TrimPrefix(s, "const "))
		if name == "" {
			return AnchorSpec{}, fmt.Errorf("invalid const anchor: %q", s)
		}
		return AnchorSpec{Kind: AnchorConst, Name: name, PrettyCached: s}, nil
	case strings.HasPrefix(s, "var "):
		name := strings.TrimSpace(strings.TrimPrefix(s, "var "))
		if name == "" {
			return AnchorSpec{}, fmt.Errorf("invalid var anchor: %q", s)
		}
		return AnchorSpec{Kind: AnchorVar, Name: name, PrettyCached: s}, nil
	default:
		return AnchorSpec{}, fmt.Errorf("invalid anchor: %q", s)
	}
}

