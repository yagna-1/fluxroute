package tenant

import (
	"fmt"
	"regexp"
	"strings"
)

var namespacePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)

func Normalize(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return "default"
	}
	return s
}

func Validate(ns string) error {
	ns = Normalize(ns)
	if !namespacePattern.MatchString(ns) {
		return fmt.Errorf("invalid namespace %q", ns)
	}
	return nil
}
