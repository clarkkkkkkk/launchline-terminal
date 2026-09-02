package launcher

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/launchline/launchline/internal/app"
)

type starterFunc func(context.Context, string, ...string) error

// PlatformLauncher isolates operating-system process details from the domain
// and UI layers.
type PlatformLauncher struct {
	goos  string
	start starterFunc
	run   starterFunc
	look  func(string) (string, error)
	stat  func(string) (os.FileInfo, error)
}

func New() *PlatformLauncher { return NewForOS(runtime.GOOS) }

func NewForOS(goos string) *PlatformLauncher {
	return &PlatformLauncher{goos: goos, start: startDetached, run: runCommand, look: exec.LookPath, stat: os.Stat}
}

func (l *PlatformLauncher) Launch(ctx context.Context, application app.Application) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("launch canceled: %w", err)
	}
	name, args, validateTarget, err := buildCommand(l.goos, application)
	if err != nil {
		return err
	}
	if validateTarget {
		if err := l.validateExecutable(application.Path); err != nil {
			return fmt.Errorf("could not launch %s: %w", application.Name, err)
		}
	} else {
		if err := l.validateOpenedTarget(name, application.Path); err != nil {
			return fmt.Errorf("could not launch %s: %w", application.Name, err)
		}
	}
	start := l.start
	if !validateTarget {
		start = l.run
	}
	if err := start(ctx, name, args...); err != nil {
		return fmt.Errorf("could not launch %s using %q: %w", application.Name, name, err)
	}
	return nil
}

func (l *PlatformLauncher) validateOpenedTarget(opener, target string) error {
	if _, err := l.look(opener); err != nil {
		return fmt.Errorf("required platform opener %q was not found", opener)
	}
	if isURL(target) {
		return nil
	}
	info, err := l.stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("application target not found: %s", target)
		}
		return fmt.Errorf("inspect application target %s: %w", target, err)
	}
	if l.goos == "linux" && info.IsDir() {
		return fmt.Errorf("desktop target is a directory: %s", target)
	}
	return nil
}

func (l *PlatformLauncher) validateExecutable(target string) error {
	if filepath.IsAbs(target) || strings.ContainsAny(target, `/\\`) {
		info, err := l.stat(target)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("executable not found: %s", target)
			}
			return fmt.Errorf("inspect executable %s: %w", target, err)
		}
		if info.IsDir() {
			return fmt.Errorf("executable path is a directory: %s", target)
		}
		return nil
	}
	if _, err := l.look(target); err != nil {
		return fmt.Errorf("executable %q was not found on PATH", target)
	}
	return nil
}

func buildCommand(goos string, application app.Application) (string, []string, bool, error) {
	target := strings.TrimSpace(application.Path)
	if target == "" {
		return "", nil, false, errors.New("application path is empty; edit it in Applications")
	}
	switch goos {
	case "windows":
		if isURL(target) {
			if len(application.Arguments) > 0 {
				return "", nil, false, errors.New("URL targets on Windows do not accept launch arguments")
			}
			return "rundll32.exe", []string{"url.dll,FileProtocolHandler", target}, false, nil
		}
		return target, append([]string(nil), application.Arguments...), true, nil
	case "darwin":
		if strings.HasSuffix(strings.ToLower(target), ".app") || isURL(target) {
			args := []string{target}
			if len(application.Arguments) > 0 {
				args = append(args, "--args")
				args = append(args, application.Arguments...)
			}
			return "open", args, false, nil
		}
		return target, append([]string(nil), application.Arguments...), true, nil
	case "linux":
		if isURL(target) || strings.HasSuffix(strings.ToLower(target), ".desktop") {
			if len(application.Arguments) > 0 {
				return "", nil, false, errors.New("Linux desktop and URL targets do not accept launch arguments; register the executable path to pass arguments")
			}
			return "xdg-open", []string{target}, false, nil
		}
		return target, append([]string(nil), application.Arguments...), true, nil
	default:
		return "", nil, false, fmt.Errorf("unsupported platform %q; Launchline supports Windows, Linux, and macOS", goos)
	}
}

func isURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != "" && (parsed.Scheme == "http" || parsed.Scheme == "https" || parsed.Scheme == "file")
}

func startDetached(_ context.Context, name string, args ...string) error {
	command := exec.Command(name, args...)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

func runCommand(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}
