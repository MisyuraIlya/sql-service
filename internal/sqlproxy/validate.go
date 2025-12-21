package sqlproxy

import (
	"fmt"
	"strings"
	"unicode"
)

func ValidateQueryReadOnly(q string) error {
	s := strings.TrimSpace(q)
	if s == "" {
		return fmt.Errorf("query is required")
	}

	low := strings.ToLower(s)

	if !(strings.HasPrefix(low, "select") || strings.HasPrefix(low, "with")) {
		return fmt.Errorf("only SELECT/WITH queries are allowed")
	}

	if strings.Contains(low, ";") {
		return fmt.Errorf("semicolon is not allowed (multi-statement blocked)")
	}
	if strings.Contains(low, "--") || strings.Contains(low, "/*") || strings.Contains(low, "*/") {
		return fmt.Errorf("sql comments are not allowed")
	}

	blocked := []string{
		"insert ", "update ", "delete ", "merge ", "truncate ",
		"drop ", "alter ", "create ", "grant ", "revoke ",
		"exec ", "execute ",
		"backup ", "restore ",
		"dbcc ",
		"xp_", "sp_",
		"openrowset", "opendatasource", "bulk ",
	}

	for _, b := range blocked {
		if strings.Contains(low, b) {
			return fmt.Errorf("blocked keyword detected: %q", strings.TrimSpace(b))
		}
	}

	return nil
}

func ValidateParamName(name string) error {
	if name == "" {
		return fmt.Errorf("param name is empty")
	}
	for i, r := range name {
		if i == 0 {
			if !(unicode.IsLetter(r) || r == '_') {
				return fmt.Errorf("param %q must start with letter/_", name)
			}
		} else {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
				return fmt.Errorf("param %q has invalid char", name)
			}
		}
	}
	return nil
}
