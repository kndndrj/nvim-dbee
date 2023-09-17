package conn

import (
	"os"
	"regexp"
)

// templateRegex matches jinja-style placeholders
var templateRegex = regexp.MustCompile(`{{.*}}`)

// envRegex matches a specific environment variable within the jinja placeholder
var envRegex = regexp.MustCompile(`env\.(\w*)`)

func expand(value string) string {
	replacer := func(sub string) string {
		matches := envRegex.FindStringSubmatch(sub)
		if len(matches) < 2 {
			return ""
		}
		return os.Getenv(matches[1])
	}

	return templateRegex.ReplaceAllStringFunc(value, replacer)
}

