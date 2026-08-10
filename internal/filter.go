package internal

import (
	"regexp"
	"strings"
)

// LineFilter removes lines from a string that match any of the given patterns.
// This is analogous to diff's --ignore-matching-lines (-I) option.
type LineFilter struct {
	patterns []*regexp.Regexp
}

// NewLineFilter creates a [LineFilter] from a slice of regular expression strings.
// Returns an error if any pattern fails to compile.
func NewLineFilter(patterns []string) (*LineFilter, error) {
	rs := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		rs[i] = r
	}
	return &LineFilter{patterns: rs}, nil
}

// Filter removes lines matching any of the compiled patterns from s.
// Lines are split by newline; the trailing newline (if any) is preserved.
func (f *LineFilter) Filter(s string) string {
	if len(f.patterns) == 0 {
		return s
	}
	// Preserve the trailing newline behaviour of the original string.
	trailingNewline := strings.HasSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	// When the string ends with "\n", Split produces an empty string as the
	// last element — exclude it from filtering so we don't accidentally drop it.
	end := len(lines)
	if trailingNewline && end > 0 {
		end--
	}

	result := lines[:0:0] // reuse backing array, length 0
	for _, line := range lines[:end] {
		if !f.matchesAny(line) {
			result = append(result, line)
		}
	}
	if trailingNewline {
		result = append(result, "")
	}
	return strings.Join(result, "\n")
}

func (f *LineFilter) matchesAny(line string) bool {
	for _, r := range f.patterns {
		if r.MatchString(line) {
			return true
		}
	}
	return false
}
