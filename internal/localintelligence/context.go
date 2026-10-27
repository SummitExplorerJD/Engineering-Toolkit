package localintelligence

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	MaxExcerpts          = 20
	MaxExcerptBytes      = 8 << 10
	MaxCombinedTextBytes = 128 << 10
)

var secretLine = regexp.MustCompile(`(?i)^\s*(token|secret|password|passwd|api_key|apikey|access_key|private_key|credential)\s*[:=]`)
var privateKeyMarker = regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)

func FilterExcerpts(input []Excerpt) []Excerpt {
	items := append([]Excerpt(nil), input...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	result := make([]Excerpt, 0, min(len(items), MaxExcerpts))
	total := 0
	for _, excerpt := range items {
		if len(result) == MaxExcerpts || !allowedPath(excerpt.Path) || ValidateDigest(excerpt.SHA256) != nil || secretBearing(excerpt.Text) {
			continue
		}
		text := truncateUTF8(excerpt.Text, MaxExcerptBytes)
		remaining := MaxCombinedTextBytes - total
		if remaining <= 0 {
			break
		}
		text = truncateUTF8(text, remaining)
		if text == "" {
			continue
		}
		excerpt.Text = text
		result = append(result, excerpt)
		total += len([]byte(text))
	}
	return result
}

func allowedPath(value string) bool {
	if value == "README.md" || value == "AGENTS.md" {
		return true
	}
	clean := path.Clean(value)
	if clean != value || strings.HasPrefix(clean, "../") {
		return false
	}
	lower := strings.ToLower(clean)
	return strings.HasSuffix(lower, ".md") && (strings.HasPrefix(clean, "docs/") || strings.HasPrefix(clean, ".agents/conventions/"))
}

func secretBearing(text string) bool {
	if privateKeyMarker.MatchString(text) {
		return true
	}
	for _, line := range strings.Split(text, "\n") {
		if secretLine.MatchString(line) {
			return true
		}
	}
	return false
}

func truncateUTF8(value string, limit int) string {
	data := []byte(value)
	if len(data) <= limit {
		return value
	}
	data = data[:limit]
	for len(data) > 0 && !utf8.Valid(data) {
		data = data[:len(data)-1]
	}
	return string(data)
}
