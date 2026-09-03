package command

import (
	"fmt"
	"sort"
	"strings"
)

type Registry struct {
	definitions []Definition
	byName      map[string]Definition
}

func NewRegistry() Registry {
	definitions := []Definition{
		{Name: "/start", Usage: "/start [workspace]", Description: "Start the default or named workspace", Action: ActionStart, MaxArgs: 1},
		{Name: "/applications", Aliases: []string{"/apps"}, Usage: "/applications", Description: "Browse discovered applications", Action: ActionApplications},
		{Name: "/workspaces", Usage: "/workspaces", Description: "Manage workspaces", Action: ActionWorkspaces},
		{Name: "/workspace", Usage: "/workspace <name>", Description: "Open a workspace", Action: ActionWorkspace, MinArgs: 1, MaxArgs: 1},
		{Name: "/add", Usage: "/add", Description: "Register an application manually", Action: ActionAdd},
		{Name: "/refresh", Usage: "/refresh", Description: "Rescan installed applications", Action: ActionRefresh},
		{Name: "/settings", Usage: "/settings", Description: "Open Launchline settings", Action: ActionSettings},
		{Name: "/help", Aliases: []string{"?"}, Usage: "/help", Description: "Show interactive commands", Action: ActionHelp},
		{Name: "/version", Usage: "/version", Description: "Show Launchline version", Action: ActionVersion},
		{Name: "/clear", Usage: "/clear", Description: "Clear session messages", Action: ActionClear},
		{Name: "/exit", Usage: "/exit", Description: "Exit Launchline", Action: ActionExit},
	}
	registry := Registry{definitions: definitions, byName: map[string]Definition{}}
	for _, definition := range definitions {
		registry.byName[strings.ToLower(definition.Name)] = definition
		for _, alias := range definition.Aliases {
			registry.byName[strings.ToLower(alias)] = definition
		}
	}
	return registry
}

func (r Registry) Definitions() []Definition { return append([]Definition(nil), r.definitions...) }

func (r Registry) Parse(value string) (Invocation, error) {
	words, err := ParseWords(value)
	if err != nil {
		return Invocation{}, err
	}
	if len(words) == 0 {
		return Invocation{}, nil
	}
	definition, ok := r.byName[strings.ToLower(words[0])]
	if !ok {
		suggestion := r.Suggest(words[0])
		if suggestion != "" {
			return Invocation{}, fmt.Errorf("unknown command: %s\nDid you mean: %s\nType /help for all commands", words[0], suggestion)
		}
		return Invocation{}, fmt.Errorf("unknown command: %s\nType /help for all commands", words[0])
	}
	argumentCount := len(words) - 1
	if argumentCount < definition.MinArgs || argumentCount > definition.MaxArgs {
		return Invocation{}, fmt.Errorf("usage: %s", definition.Usage)
	}
	return Invocation{Definition: definition, Arguments: words[1:], Raw: strings.TrimSpace(value)}, nil
}

func (r Registry) Complete(value string, workspaceNames []string) (string, []string) {
	trimmedLeft := strings.TrimLeft(value, " \t")
	if trimmedLeft == "" {
		return value, nil
	}
	words, _ := ParseWords(trimmedLeft)
	if !strings.Contains(trimmedLeft, " ") {
		prefix := strings.ToLower(trimmedLeft)
		if !strings.HasPrefix(prefix, "/") && prefix != "?" {
			prefix = "/" + prefix
		}
		// Prefer the plural management surface while completing the command
		// name. The singular form remains available with an explicit argument.
		if prefix == "/work" || prefix == "/workspace" {
			return "/workspaces", []string{"/workspaces"}
		}
		if definition, ok := r.byName[prefix]; ok {
			completed := definition.Name
			if definition.MaxArgs > 0 {
				completed += " "
			}
			return completed, []string{definition.Name}
		}
		var matches []string
		seen := map[string]bool{}
		for name, definition := range r.byName {
			if strings.HasPrefix(name, prefix) && !seen[definition.Name] {
				matches = append(matches, definition.Name)
				seen[definition.Name] = true
			}
		}
		sort.Strings(matches)
		if len(matches) == 1 {
			return matches[0], matches
		}
		if common := commonPrefix(matches); len(common) > len(prefix) {
			return common, matches
		}
		return value, matches
	}
	if len(words) >= 1 {
		definition, ok := r.byName[strings.ToLower(words[0])]
		if ok && (definition.Action == ActionStart || definition.Action == ActionWorkspace) {
			prefix := ""
			if len(words) > 1 {
				prefix = strings.ToLower(words[len(words)-1])
			}
			var matches []string
			for _, name := range workspaceNames {
				if strings.HasPrefix(strings.ToLower(name), prefix) {
					matches = append(matches, name)
				}
			}
			sort.Strings(matches)
			if len(matches) == 1 {
				completed := matches[0]
				if strings.ContainsAny(completed, " \t") {
					completed = `"` + strings.ReplaceAll(completed, `"`, `\"`) + `"`
				}
				return definition.Name + " " + completed, matches
			}
			return value, matches
		}
	}
	return value, nil
}

func commonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, prefix) && prefix != "" {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func (r Registry) Suggest(value string) string {
	value = strings.ToLower(value)
	best, distance := "", 4
	for _, definition := range r.definitions {
		d := editDistance(value, definition.Name)
		if d < distance {
			best, distance = definition.Name, d
		}
	}
	if distance <= 3 {
		return best
	}
	return ""
}

func editDistance(a, b string) int {
	previous := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, ar := range []rune(a) {
		current := make([]int, len(previous))
		current[0] = i + 1
		for j, br := range []rune(b) {
			cost := 0
			if ar != br {
				cost = 1
			}
			current[j+1] = min3(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(previous)-1]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
