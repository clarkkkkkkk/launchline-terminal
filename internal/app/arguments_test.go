package app

import (
	"reflect"
	"testing"
)

func TestArgumentInputCases(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  []string
	}{
		{"", nil}, {" \t ", nil}, {"--incognito", []string{"--incognito"}},
		{". --classic", []string{".", "--classic"}},
		{`--profile "Work Profile"`, []string{"--profile", "Work Profile"}},
		{`'' ""`, []string{"", ""}},
		{`"C:\Program Files\Editor"`, []string{`C:\Program Files\Editor`}},
		{`&& || ; | > $(echo) $HOME ~`, []string{"&&", "||", ";", "|", ">", "$(echo)", "$HOME", "~"}},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseArguments(tt.input)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}
	for _, input := range []string{`"unclosed`, `'unclosed`} {
		if _, err := ParseArguments(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}

func TestArgumentFormatRoundtrip(t *testing.T) {
	args := []string{"", ".", "--classic", "Work Profile", "a\nb", "a\rb", "a\u2003b", `a"b`, `it's`, `C:\Program Files\`, `\\server\share`, `\\server\Work Share\`, `a\"b`, `a\\b`, `;`, `$(touch nope)`}
	got, err := ParseArguments(FormatArguments(args))
	if err != nil || !reflect.DeepEqual(got, args) {
		t.Fatalf("roundtrip: %#v, %v; want %#v", got, err, args)
	}
}

func FuzzArgumentFormatRoundtrip(f *testing.F) {
	for _, arg := range []string{"", `C:\Program Files\`, `\\server\share`, "a\u2003b", `a"b`} {
		f.Add(arg)
	}
	f.Fuzz(func(t *testing.T, arg string) {
		// The input parser operates on Unicode text.
		normalized := string([]rune(arg))
		got, err := ParseArguments(FormatArguments([]string{normalized}))
		if err != nil || len(got) != 1 || got[0] != normalized {
			t.Fatalf("%q => %#v, %v", normalized, got, err)
		}
	})
}
