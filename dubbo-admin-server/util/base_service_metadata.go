package util

import "strings"

func BuildServiceKey(path, group, version string) string {
	var length int
	if path != "" {
		length += len(path)
	}
	if group != "" {
		length += len(group)
	}
	if version != "" {
		length += len(version)
	}
	length += 2

	var buf strings.Builder
	buf.Grow(length)

	if group != "" {
		buf.WriteString(group)
		buf.WriteString("/")
	}
	buf.WriteString(path)
	if version != "" {
		buf.WriteString(":")
		buf.WriteString(version)
	}

	return buf.String()
}
