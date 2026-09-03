package command

import (
	"errors"
	"strings"
	"unicode"
)

func ParseWords(value string) ([]string, error) {
	var words []string
	var current strings.Builder
	var quote rune
	started := false
	flush := func() {
		if started {
			words = append(words, current.String())
			current.Reset()
			started = false
		}
	}
	runes := []rune(strings.TrimSpace(value))
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote != 0 {
			if r == quote {
				quote = 0
			} else if r == '\\' && i+1 < len(runes) && (runes[i+1] == quote || runes[i+1] == '\\') {
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
		} else if unicode.IsSpace(r) {
			flush()
		} else if r == '\\' && i+1 < len(runes) && (unicode.IsSpace(runes[i+1]) || runes[i+1] == '\\' || runes[i+1] == '\'' || runes[i+1] == '"') {
			i++
			current.WriteRune(runes[i])
			started = true
		} else {
			current.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		return nil, errors.New("command contains an unclosed quote")
	}
	flush()
	return words, nil
}
