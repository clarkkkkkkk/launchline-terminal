package assets

import (
	_ "embed"
	"strings"
)

// LaunchlineLogo is the canonical large terminal wordmark.
//
//go:embed ascii/launchline.txt
var launchlineLogo string

func LaunchlineLogo() string {
	return strings.TrimRight(strings.ReplaceAll(launchlineLogo, "\r\n", "\n"), "\r\n")
}
