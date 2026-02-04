package common

import "strings"

func SummarizeText(text string) string {
	value := strings.TrimSpace(text)
	if value == "" {
		return ""
	}
	const limit = 120
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
