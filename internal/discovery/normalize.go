package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

func StableID(platform, kind, target string, _ []string) string {
	canonical := strings.ToLower(strings.TrimSpace(target))
	if platform != "windows" {
		canonical = filepath.Clean(strings.TrimSpace(target))
	}
	value := strings.Join([]string{platform, kind, canonical}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return "discovered_" + hex.EncodeToString(sum[:12])
}

// Normalize cleans, identifies, sorts, and deterministically deduplicates
// launch targets. Name similarity alone is never used to merge applications.
func Normalize(items []Application) []Application {
	byID := make(map[string]Application, len(items))
	for _, item := range items {
		item.Name = strings.TrimSpace(item.Name)
		item.Target = strings.TrimSpace(item.Target)
		item.Source = strings.TrimSpace(item.Source)
		if item.Platform == "" {
			item.Platform = platformName()
		}
		if item.Kind == "" {
			item.Kind = KindExecutable
		}
		if item.Name == "" || item.Target == "" {
			continue
		}
		item.ID = StableID(item.Platform, item.Kind, item.Target, item.Arguments)
		if current, exists := byID[item.ID]; !exists || sourceRank(item.Source) < sourceRank(current.Source) {
			byID[item.ID] = item
		}
	}
	out := make([]Application, 0, len(byID))
	for _, item := range byID {
		out = append(out, item)
	}
	sortApplications(out)
	return out
}

func sourceRank(source string) int {
	switch source {
	case "windows-app-paths", "macos-bundle", "xdg-desktop":
		return 0
	case "windows-start-menu":
		return 1
	default:
		return 2
	}
}
