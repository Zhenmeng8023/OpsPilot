package security

import (
	"regexp"
	"strings"
)

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(bearer\s+)([A-Za-z0-9._~+/\-=]+)`),
	regexp.MustCompile(`(?i)(([A-Za-z0-9_.-]*(token|password|secret|authorization)[A-Za-z0-9_.-]*)\s*[:=]\s*)([^\s,;}]+)`),
}

var sensitiveJSONPattern = regexp.MustCompile(`(?i)("[^"]*(token|password|secret|authorization)[^"]*"\s*:\s*)("[^"]*"|[0-9]+|true|false|null)`)

func Redact(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	redacted := value
	for _, pattern := range sensitivePatterns {
		redacted = pattern.ReplaceAllString(redacted, `${1}***`)
	}
	redacted = sensitiveJSONPattern.ReplaceAllString(redacted, `${1}"***"`)
	return redacted
}
