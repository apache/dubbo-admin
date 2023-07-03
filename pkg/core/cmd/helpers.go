package cmd

import (
	"fmt"
	"strings"
)

func UsageOptions(desc string, options ...interface{}) string {
	values := make([]string, 0, len(options))
	for _, option := range options {
		values = append(values, fmt.Sprintf("%v", option))
	}
	return fmt.Sprintf("%s: one of %s", desc, strings.Join(values, "|"))
}
