//go:build windows

package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func newPlatformDiscoverer() Discoverer {
	return multiDiscoverer{sources: []namedSource{
		{name: "Windows Start Menu", find: DiscovererFunc(discoverWindowsStartMenu)},
		{name: "Windows App Paths", find: DiscovererFunc(discoverWindowsAppPaths)},
	}}
}

func discoverWindowsStartMenu(ctx context.Context) ([]Application, error) {
	var roots []string
	if appData := os.Getenv("APPDATA"); appData != "" {
		roots = append(roots, filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		roots = append(roots, filepath.Join(programData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	var items []Application
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if path == root {
					return err
				}
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".lnk") {
				return nil
			}
			items = append(items, windowsShortcutApplication(path))
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return items, err
		}
	}
	return items, nil
}

func discoverWindowsAppPaths(ctx context.Context) ([]Application, error) {
	const keyPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`
	var items []Application
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		key, err := registry.OpenKey(hive, keyPath, registry.READ)
		if err != nil {
			continue
		}
		names, err := key.ReadSubKeyNames(-1)
		_ = key.Close()
		if err != nil {
			continue
		}
		for _, subName := range names {
			if err := ctx.Err(); err != nil {
				return items, err
			}
			sub, err := registry.OpenKey(hive, keyPath+`\`+subName, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			target, _, err := sub.GetStringValue("")
			_ = sub.Close()
			if err != nil || strings.TrimSpace(target) == "" {
				continue
			}
			name := strings.TrimSuffix(filepath.Base(subName), filepath.Ext(subName))
			items = append(items, Application{Name: name, Target: target, Kind: KindExecutable, Source: "windows-app-paths", Platform: "windows"})
		}
	}
	return items, nil
}
