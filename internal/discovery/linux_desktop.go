package discovery

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type desktopEntry struct {
	Name, Exec, EntryType       string
	Hidden, NoDisplay, Terminal bool
}

func parseDesktopEntry(reader io.Reader) (desktopEntry, error) {
	entry := desktopEntry{}
	inDesktopEntry := false
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inDesktopEntry = line == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntry {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "Name":
			entry.Name = value
		case "Exec":
			entry.Exec = value
		case "Type":
			entry.EntryType = value
		case "Hidden":
			entry.Hidden = strings.EqualFold(value, "true")
		case "NoDisplay":
			entry.NoDisplay = strings.EqualFold(value, "true")
		case "Terminal":
			entry.Terminal = strings.EqualFold(value, "true")
		}
	}
	if err := scanner.Err(); err != nil {
		return entry, err
	}
	if entry.EntryType != "Application" || entry.Hidden || entry.NoDisplay || entry.Terminal || strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Exec) == "" {
		return entry, errors.New("desktop entry is not a visible graphical application")
	}
	return entry, nil
}

func parseDesktopExec(value string) (string, []string, error) {
	var values []string
	var current strings.Builder
	var quote rune
	started := false
	flush := func() {
		if started {
			values = append(values, current.String())
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
			} else if r == '\\' && i+1 < len(runes) {
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
		} else if r == '\\' && i+1 < len(runes) {
			i++
			current.WriteRune(runes[i])
			started = true
		} else if r == ' ' || r == '\t' {
			flush()
		} else {
			current.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		return "", nil, errors.New("Exec contains an unclosed quote")
	}
	flush()
	clean := make([]string, 0, len(values))
	for _, item := range values {
		if item == "%f" || item == "%F" || item == "%u" || item == "%U" || item == "%i" || item == "%c" || item == "%k" {
			continue
		}
		item = strings.ReplaceAll(item, "%%", "\x00")
		for _, code := range []string{"%f", "%F", "%u", "%U", "%i", "%c", "%k"} {
			item = strings.ReplaceAll(item, code, "")
		}
		item = strings.ReplaceAll(item, "\x00", "%")
		if item != "" {
			clean = append(clean, item)
		}
	}
	if len(clean) == 0 {
		return "", nil, errors.New("Exec has no executable")
	}
	if clean[0] == "env" {
		return "", nil, fmt.Errorf("Exec environment wrappers are not supported safely")
	}
	return clean[0], clean[1:], nil
}
