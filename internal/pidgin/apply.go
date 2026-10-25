package pidgin

import (
	"fmt"
	"strings"
)

// MatchError describes a whitespace-tolerant matching failure during hunk apply.
type MatchError struct {
	Kind       string // "context" or "delete"
	Line       int    // 1-based line number in base
	Expected   string // leading-WS-trimmed expected
	Found      string // leading-WS-trimmed found
	Paragraph  string // cached one-line message
}

func (e *MatchError) Error() string {
	if e.Paragraph != "" {
		return e.Paragraph
	}
	msg := fmt.Sprintf("%s mismatch at line %d: expected %q, got %q (hint: re-copy line from file)", e.Kind, e.Line, strings.TrimLeft(e.Expected, " \t"), strings.TrimLeft(e.Found, " \t"))
	e.Paragraph = msg
	return msg
}

// ApplyUnified applies a simple unified hunk (context/-/+) onto base.
// It supports a single contiguous region per hunk, with optional 0-2 context lines.
// When a hunk contains only '+' lines (no context or '-'), it REPLACES the entire base span.
func ApplyUnified(base string, diff []DiffLine) (string, error) {
	baseLines := splitKeepNL(base)
	// Build the sequence of lines that must match in order (contexts and dels)
	expected := make([]string, 0, len(diff))
	addOnly := true
	addBuf := make([]string, 0, len(diff))
	for _, dl := range diff {
		if dl.Kind == LineContext || dl.Kind == LineDel {
			expected = append(expected, dl.Text+"\n")
			addOnly = false
		}
		if dl.Kind == LineAdd {
			addBuf = append(addBuf, dl.Text+"\n")
		}
	}
	// Only additions: replace entire base span
	if addOnly {
		if len(addBuf) == 0 {
			return "", fmt.Errorf("empty replacement: only '+' lines requested but none provided")
		}
		return strings.Join(addBuf, ""), nil
	}
	start := 0
	if len(expected) > 0 {
		pos, failBaseIdx, failNeedleIdx := findSubsequenceDetailed(baseLines, expected)
		if pos < 0 {
			// Could not locate context region; if we have a concrete mismatch candidate, surface it.
			if failNeedleIdx >= 0 && failBaseIdx >= 0 {
				return "", &MatchError{
					Kind:     "context",
					Line:     indexToLine(baseLines, failBaseIdx),
					Expected: expected[failNeedleIdx],
					Found:    baseLines[failBaseIdx],
				}
			}
			// Otherwise surface the first expected line
			first := ""
			if len(expected) > 0 {
				first = expected[0]
			}
			return "", &MatchError{
				Kind:     "context",
				Line:     0,
				Expected: first,
				Found:    "",
			}
		}
		start = pos
		if failNeedleIdx >= 0 && failBaseIdx >= 0 {
			return "", &MatchError{
				Kind:     "context",
				Line:     indexToLine(baseLines, failBaseIdx),
				Expected: expected[failNeedleIdx],
				Found:    baseLines[failBaseIdx],
			}
		}
	}
	// Rebuild with patch region applied
	var out []string
	out = append(out, baseLines[:start]...)
	i := start
	for _, dl := range diff {
		switch dl.Kind {
		case LineAdd:
			out = append(out, dl.Text+"\n")
		case LineContext:
			// Must match next base line
			if i >= len(baseLines) || !eqLine(baseLines[i], dl.Text+"\n") {
				return "", &MatchError{
					Kind:     "context",
					Line:     indexToLine(baseLines, i),
					Expected: dl.Text + "\n",
					Found:    guardIndex(baseLines, i),
				}
			}
			out = append(out, baseLines[i])
			i++
		case LineDel:
			// Must match next base line; then skip
			if i >= len(baseLines) || !eqLine(baseLines[i], dl.Text+"\n") {
				return "", &MatchError{
					Kind:     "delete",
					Line:     indexToLine(baseLines, i),
					Expected: dl.Text + "\n",
					Found:    guardIndex(baseLines, i),
				}
			}
			i++
		}
	}
	// Remainder
	out = append(out, baseLines[i:]...)
	return strings.Join(out, ""), nil
}

func splitKeepNL(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	// strings.SplitAfter keeps trailing empty for no trailing newline; keep as-is
	return parts
}

func findSubsequenceDetailed(haystack []string, needle []string) (pos int, failBaseIdx int, failNeedleIdx int) {
	if len(needle) == 0 {
		return 0, -1, -1
	}
	firstFailBase := -1
	firstFailNeedle := -1
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if !eqLine(haystack[i], needle[0]) {
			continue
		}
		// candidate
		mismatch := -1
		for j := 0; j < len(needle); j++ {
			if !eqLine(haystack[i+j], needle[j]) {
				mismatch = j
				break
			}
		}
		if mismatch == -1 {
			return i, -1, -1 // success
		}
		// record first mismatch but continue scanning further candidates
		if firstFailBase < 0 && firstFailNeedle < 0 {
			firstFailBase = i + mismatch
			firstFailNeedle = mismatch
		}
		// continue to next i
	}
	// Not found at all; maybe show first expected line wasn't found
	if firstFailBase >= 0 {
		return -1, firstFailBase, firstFailNeedle
	}
	return -1, -1, -1
}

func eqLine(a, b string) bool {
	// compare ignoring leading whitespace differences and trailing line endings
	norm := func(s string) string {
		s = strings.TrimLeft(s, " \t")
		s = strings.TrimRight(s, "\r\n")
		return s
	}
	return norm(a) == norm(b)
}

func indexToLine(lines []string, idx int) int {
	// 1-based line number equals idx+1 since we split by lines
	return idx + 1
}

func guardIndex(lines []string, idx int) string {
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	return lines[idx]
}

