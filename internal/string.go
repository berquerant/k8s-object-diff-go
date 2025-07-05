package internal

import "strings"

func HeadString(s, sep string, n int) string {
	if n < 1 {
		return ""
	}
	ss := strings.Split(s, sep)
	if n >= len(ss) {
		return s
	}
	return strings.Join(ss[:n], sep) + sep
}

func TailString(s, sep string, n int) string {
	if n < 1 {
		return ""
	}
	ss := strings.Split(s, sep)
	if n >= len(ss) {
		return s
	}
	return strings.Join(ss[len(ss)-n-1:], sep)
}
