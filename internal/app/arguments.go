package app

import (
	"errors"
	"strings"
	"unicode"
)

// ParseArguments parses a small, shell-independent quoted argument format. It
// performs no expansion or evaluation; its output is passed directly to exec.
func ParseArguments(value string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	started := false
	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}
	runes := []rune(value)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote != 0 {
			if r == quote {
				quote = 0
			} else if r == '\\' && quote == '"' && i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\\') {
				i++
				current.WriteRune(runes[i])
			} else {
				current.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '\'' || r == '"' {
			quote, started = r, true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		if r == '\\' && i+1 < len(runes) && (unicode.IsSpace(runes[i+1]) || runes[i+1] == '\'' || runes[i+1] == '"' || runes[i+1] == '\\') {
			i++
			current.WriteRune(runes[i])
			started = true
			continue
		}
		current.WriteRune(r)
		started = true
	}
	if quote != 0 {
		return nil, errors.New("arguments contain an unclosed quote")
	}
	flush()
	return args, nil
}

func FormatArguments(args []string) string {
	formatted := make([]string, len(args))
	for i, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\"'") {
			formatted[i] = `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
		} else {
			formatted[i] = arg
		}
	}
	return strings.Join(formatted, " ")
}
