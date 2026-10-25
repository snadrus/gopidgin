package pidgin

import (
	"fmt"
	"strings"
)

// ApplyUnified applies a simple unified hunk (context/-/+) onto base.
// It supports a single contiguous region per hunk, with optional 0-2 context lines.
// If there are no context/del lines (pure insertion), it inserts at the end.
func ApplyUnified(base string, diff []DiffLine) (string, error) {
	baseLines := splitKeepNL(base)
	// Build the sequence of lines that must match in order (contexts and dels)
	expected := make([]string, 0, len(diff))
	for _, dl := range diff {
		if dl.Kind == LineContext || dl.Kind == LineDel {
			expected = append(expected, dl.Text+"\n")
		}
	}
	start := 0
	if len(expected) > 0 {
		pos := findSubsequence(baseLines, expected)
		if pos < 0 {
			return "", fmt.Errorf("context does not match target")
		}
		start = pos
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
				return "", fmt.Errorf("context mismatch applying at position")
			}
			out = append(out, baseLines[i])
			i++
		case LineDel:
			// Must match next base line; then skip
			if i >= len(baseLines) || !eqLine(baseLines[i], dl.Text+"\n") {
				return "", fmt.Errorf("delete mismatch applying at position")
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

func findSubsequence(haystack []string, needle []string) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		ok := true
		for j := 0; j < len(needle); j++ {
			if !eqLine(haystack[i+j], needle[j]) {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func eqLine(a, b string) bool {
	// compare ignoring leading whitespace differences
	return strings.TrimLeft(a, " \t") == strings.TrimLeft(b, " \t")
}

