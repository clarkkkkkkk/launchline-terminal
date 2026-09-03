//go:build linux

package discovery

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func newPlatformDiscoverer() Discoverer {
	return multiDiscoverer{sources: []namedSource{{name: "XDG desktop entries", find: DiscovererFunc(discoverLinuxDesktopEntries)}}}
}

func linuxApplicationDirectories() []string {
	home, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" && home != "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	roots := []string{}
	if dataHome != "" {
		roots = append(roots, filepath.Join(dataHome, "applications"))
	}
	for _, root := range filepath.SplitList(dataDirs) {
		if root != "" {
			roots = append(roots, filepath.Join(root, "applications"))
		}
	}
	// Common package exports are optional and harmless when absent.
	roots = append(roots, "/var/lib/flatpak/exports/share/applications", filepath.Join(home, ".local/share/flatpak/exports/share/applications"), "/var/lib/snapd/desktop/applications")
	return roots
}

func discoverLinuxDesktopEntries(ctx context.Context) ([]Application, error) {
	var items []Application
	var warnings []error
	seenDirs := map[string]bool{}
	for _, root := range linuxApplicationDirectories() {
		root = filepath.Clean(root)
		if seenDirs[root] {
			continue
		}
		seenDirs[root] = true
		entries, err := os.ReadDir(root)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			warnings = append(warnings, fmt.Errorf("read %s: %w", root, err))
			continue
		}
		for _, file := range entries {
			if err := ctx.Err(); err != nil {
				return items, err
			}
			if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".desktop") {
				continue
			}
			path := filepath.Join(root, file.Name())
			handle, err := os.Open(path)
			if err != nil {
				continue
			}
			entry, parseErr := parseDesktopEntry(handle)
			_ = handle.Close()
			if parseErr != nil {
				continue
			}
			target, arguments, err := parseDesktopExec(entry.Exec)
			if err != nil {
				continue
			}
			items = append(items, Application{Name: entry.Name, Target: target, Arguments: arguments, Kind: KindExecutable, Source: "xdg-desktop", Platform: "linux", Metadata: map[string]string{"desktop_file": path}})
		}
	}
	if len(warnings) > 0 {
		return items, errors.Join(warnings...)
	}
	return items, nil
}
