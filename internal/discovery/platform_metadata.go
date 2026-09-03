package discovery

import (
	"encoding/xml"
	"path/filepath"
	"strings"
)

func plistString(data []byte, key string) string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	wantValue := false
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "key" {
				var name string
				if decoder.DecodeElement(&name, &value) == nil {
					wantValue = name == key
				}
			} else if wantValue && value.Name.Local == "string" {
				var result string
				if decoder.DecodeElement(&result, &value) == nil {
					return strings.TrimSpace(result)
				}
			}
		}
	}
}

func macBundleApplication(bundle string, info []byte) Application {
	name := strings.TrimSuffix(filepath.Base(bundle), filepath.Ext(bundle))
	if display := plistString(info, "CFBundleDisplayName"); display != "" {
		name = display
	} else if bundleName := plistString(info, "CFBundleName"); bundleName != "" {
		name = bundleName
	}
	return Application{Name: name, Target: bundle, Kind: KindBundle, Source: "macos-bundle", Platform: "darwin"}
}

func windowsShortcutApplication(path string) Application {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Application{Name: name, Target: path, Kind: KindShortcut, Source: "windows-start-menu", Platform: "windows"}
}
