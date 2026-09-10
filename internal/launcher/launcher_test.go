package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
	l.start = func(context.Context, processCommand) error { t.Fatal("start called"); return nil }
	err := l.Launch(context.Background(), app.Application{Name: "Cursor", Path: "/missing/cursor"})
	if err == nil || !strings.Contains(err.Error(), "executable not found") || !strings.Contains(err.Error(), "/missing/cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMissingApplicationBundleReturnsActionableError(t *testing.T) {
	l := NewForOS("darwin")
	l.look = func(string) (string, error) { return "/usr/bin/open", nil }
	l.stat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	l.start = func(context.Context, processCommand) error { t.Fatal("start called"); return nil }
	err := l.Launch(context.Background(), app.Application{Name: "Cursor", Path: "/Applications/Cursor.app"})
	if err == nil || !strings.Contains(err.Error(), "application target not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStarterFailureIsWrapped(t *testing.T) {
	l := NewForOS("linux")
	l.look = func(string) (string, error) { return "/bin/editor", nil }
	l.start = func(context.Context, processCommand) error { return errors.New("permission denied") }
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

func TestRepresentativeLaunchCommands(t *testing.T) {
	for _, tt := range []struct {
		goos, path, name string
		input, want      []string
	}{
		{"linux", "/usr/bin/cursor", "/usr/bin/cursor", []string{".", "--classic"}, []string{".", "--classic"}},
		{"windows", `C:\Program Files\SomeApp\app.exe`, `C:\Program Files\SomeApp\app.exe`, []string{"--flag"}, []string{"--flag"}},
		{"darwin", "/Applications/SomeApp.app", "open", []string{"--flag"}, []string{"/Applications/SomeApp.app", "--args", "--flag"}},
		{"linux", "/usr/bin/firefox", "/usr/bin/firefox", nil, nil},
	} {
		name, args, _, err := buildCommand(tt.goos, app.Application{Path: tt.path, Arguments: tt.input})
		if err != nil || name != tt.name || !reflect.DeepEqual(args, tt.want) {
			t.Fatalf("%s: %s %#v, %v", tt.goos, name, args, err)
		}
	}
}

func TestWorkingDirectoryFailuresNeverStart(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{filepath.Join(t.TempDir(), "missing"), file, "bad\x00directory"} {
		l := New()
		l.start = func(context.Context, processCommand) error { t.Fatal("unexpected process start"); return nil }
		err := l.Launch(context.Background(), app.Application{Name: "Editor", Path: "editor", WorkingDirectory: directory})
		if err == nil || !strings.Contains(err.Error(), "Editor") || !strings.Contains(err.Error(), "working directory") {
			t.Fatalf("directory %q: %v", directory, err)
		}
	}
	for _, tt := range []struct{ goos, path string }{
		{"windows", `C:\Apps\Editor.lnk`}, {"windows", "https://example.com"},
		{"linux", "/apps/editor.desktop"}, {"linux", "https://example.com"},
		{"darwin", "/Applications/Editor.app"}, {"darwin", "https://example.com"},
	} {
		l := NewForOS(tt.goos)
		l.start = func(context.Context, processCommand) error { t.Fatal("unexpected start"); return nil }
		l.run = l.start
		err := l.Launch(context.Background(), app.Application{Name: "Editor", Path: tt.path, WorkingDirectory: t.TempDir()})
		if err == nil || !strings.Contains(err.Error(), "register the executable") {
			t.Fatalf("%s %s: %v", tt.goos, tt.path, err)
		}
	}
}

func TestWorkingDirectoryPreservesPATHResolution(t *testing.T) {
	target := filepath.Join(t.TempDir(), "editor")
	l := New()
	l.look = func(name string) (string, error) {
		if name != "editor" {
			t.Fatalf("lookup %q", name)
		}
		return target, nil
	}
	directory := t.TempDir()
	l.start = func(_ context.Context, spec processCommand) error {
		if spec.name != target || spec.dir != directory || !reflect.DeepEqual(spec.args, []string{"--flag"}) {
			t.Fatalf("spec: %#v", spec)
		}
		return nil
	}
	if err := l.Launch(context.Background(), app.Application{Path: "editor", Arguments: []string{"--flag"}, WorkingDirectory: directory}); err != nil {
		t.Fatal(err)
	}
}

type processReport struct {
	Args      []string
	Directory string
}

func TestLaunchProcessHelper(t *testing.T) {
	reportPath := os.Getenv("LAUNCHLINE_TEST_PROCESS_REPORT")
	if reportPath == "" {
		return
	}
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		t.Fatal("missing argument separator")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(processReport{Args: os.Args[separator+1:], Directory: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestActualProcessArgumentsAndDirectory(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	// Windows CI keeps the checkout on D: and the Go test executable/temp
	// directories on C:. Only the relative-path case needs a common volume;
	// keep the absolute-path cases in the original checkout directory.
	relativeBase, err := os.MkdirTemp(filepath.Dir(executable), "launchline-relative-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// The detached helper can briefly hold its cwd open on Windows after
		// writing the report. Allow it to exit before removing the fixture.
		deadline := time.Now().Add(5 * time.Second)
		for {
			err := os.RemoveAll(relativeBase)
			if err == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Error(err)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
	relativeDirectory := "project with spaces"
	relativeTargetDirectory := filepath.Join(relativeBase, relativeDirectory)
	if err := os.Mkdir(relativeTargetDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	relativeExecutable, err := filepath.Rel(relativeBase, executable)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, target, directory, wantDirectory, launchDirectory string
		args                                                    []string
	}{
		{"default-no-args", executable, "", cwd, "", []string{}},
		{"custom", executable, directory, directory, "", []string{".", "--classic", "Work Profile", "", `C:\Work\`, "&&", ";", "$(echo nope)"}},
		{"relative", relativeExecutable, relativeDirectory, relativeTargetDirectory, relativeBase, []string{"--flag"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.launchDirectory != "" {
				t.Chdir(tt.launchDirectory)
			}
			launchDirectory, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			reportPath := filepath.Join(t.TempDir(), "report.json")
			t.Setenv("LAUNCHLINE_TEST_PROCESS_REPORT", reportPath)
			args := append([]string{"-test.run=^TestLaunchProcessHelper$", "--"}, tt.args...)
			if err := New().Launch(context.Background(), app.Application{Name: "Helper", Path: tt.target, Arguments: args, WorkingDirectory: tt.directory}); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			var report processReport
			for {
				data, err := os.ReadFile(reportPath)
				if err == nil && json.Unmarshal(data, &report) == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("helper did not report: %v", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
			gotDir, err := os.Stat(report.Directory)
			if err != nil {
				t.Fatal(err)
			}
			wantDir, err := os.Stat(tt.wantDirectory)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(gotDir, wantDir) || !reflect.DeepEqual(report.Args, tt.args) {
				t.Fatalf("received %#v; want directory %q args %#v", report, tt.wantDirectory, tt.args)
			}
			after, err := os.Getwd()
			if err != nil || after != launchDirectory {
				t.Fatal("Launchline cwd changed")
			}
		})
	}
}
