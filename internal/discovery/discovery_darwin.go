//go:build darwin

package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func newPlatformDiscoverer() Discoverer {
	return multiDiscoverer{sources: []namedSource{{name: "macOS application bundles", find: DiscovererFunc(discoverMacApplications)}}}
}

func discoverMacApplications(ctx context.Context) ([]Application, error) {
	home, _ := os.UserHomeDir()
	roots := []string{"/Applications", filepath.Join(home, "Applications"), "/System/Applications"}
	var items []Application
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if path == root {
					return walkErr
				}
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if !entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
				return nil
			}
			data, _ := os.ReadFile(filepath.Join(path, "Contents", "Info.plist"))
			items = append(items, macBundleApplication(path, data))
			return fs.SkipDir
		})
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return items, err
		}
	}
	return items, nil
}
