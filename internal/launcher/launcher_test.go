package launcher

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/launchline/launchline/internal/app"
)

func TestBuildCommandsDoNotConcatenateArguments(t *testing.T) {
	application := app.Application{Name: "Editor", Path: `/opt/My Editor/editor`, Arguments: []string{"--profile", "work; rm -rf nope"}}
	tests := []struct {
		goos string
		name string
		args []string
	}{
		{"linux", application.Path, application.Arguments},
		{"darwin", application.Path, application.Arguments},
		{"windows", application.Path, application.Arguments},
	}
	for _, test := range tests {
		name, args, validate, err := buildCommand(test.goos, application)
		if err != nil {
			t.Fatal(err)
		}
		if name != test.name || !reflect.DeepEqual(args, test.args) || !validate {
			t.Fatalf("%s: %q %#v validate=%v", test.goos, name, args, validate)
		}
	}
}

func TestPlatformSpecificTargets(t *testing.T) {
	name, args, validate, err := buildCommand("darwin", app.Application{Path: "/Applications/Cursor.app", Arguments: []string{"--new-window"}})
	if err != nil || name != "open" || validate || !reflect.DeepEqual(args, []string{"/Applications/Cursor.app", "--args", "--new-window"}) {
		t.Fatalf("macOS command: %q %#v %v %v", name, args, validate, err)
	}
	name, args, validate, err = buildCommand("linux", app.Application{Path: "https://example.com"})
	if err != nil || name != "xdg-open" || validate || !reflect.DeepEqual(args, []string{"https://example.com"}) {
		t.Fatalf("Linux URL command: %q %#v %v %v", name, args, validate, err)
	}
}

func TestLinuxURLRejectsArguments(t *testing.T) {
	_, _, _, err := buildCommand("linux", app.Application{Path: "https://example.com", Arguments: []string{"unexpected"}})
	if err == nil || !strings.Contains(err.Error(), "do not accept launch arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMissingExecutableReturnsActionableError(t *testing.T) {
	l := NewForOS("linux")
	l.stat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	l.start = func(context.Context, string, ...string) error { t.Fatal("start called"); return nil }
	err := l.Launch(context.Background(), app.Application{Name: "Cursor", Path: "/missing/cursor"})
	if err == nil || !strings.Contains(err.Error(), "executable not found") || !strings.Contains(err.Error(), "/missing/cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMissingApplicationBundleReturnsActionableError(t *testing.T) {
	l := NewForOS("darwin")
	l.look = func(string) (string, error) { return "/usr/bin/open", nil }
	l.stat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	l.start = func(context.Context, string, ...string) error { t.Fatal("start called"); return nil }
	err := l.Launch(context.Background(), app.Application{Name: "Cursor", Path: "/Applications/Cursor.app"})
	if err == nil || !strings.Contains(err.Error(), "application target not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStarterFailureIsWrapped(t *testing.T) {
	l := NewForOS("linux")
	l.look = func(string) (string, error) { return "/bin/editor", nil }
	l.start = func(context.Context, string, ...string) error { return errors.New("permission denied") }
	err := l.Launch(context.Background(), app.Application{Name: "Editor", Path: "editor"})
	if err == nil || !strings.Contains(err.Error(), "permission denied") || !strings.Contains(err.Error(), "Editor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnsupportedPlatform(t *testing.T) {
	_, _, _, err := buildCommand("plan9", app.Application{Path: "editor"})
	if err == nil || !strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("unexpected error: %v", err)
	}
}
