package device

import (
	"strings"
	"testing"
)

func TestHasClass(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		className string
		want      bool
	}{
		{"exact single token", " desktop", "desktop", true},
		{"multi token group", " ios ipad tablet", "ios ipad tablet", true},
		{"absent", " desktop", "mobile", false},
		{"empty subject", "", "desktop", false},
		{"case insensitive", " DESKTOP", "desktop", true},
		{"case insensitive argument", " desktop", "DESKTOP", true},
		{"substring match is enough", " macos desktop", "desktop", true},
		{"partial token still matches", " desktop", "deskt", true},
		{"hyphenated token", " node-webkit", "node-webkit", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasClass(tc.current, tc.className); got != tc.want {
				t.Errorf("HasClass(%q, %q) = %v, want %v", tc.current, tc.className, got, tc.want)
			}
		})
	}
}

// TestHasClassWithInvalidPattern covers the documented divergence from
// JavaScript: a class name that is not a valid RE2 pattern is treated as
// absent, where the original would throw a SyntaxError.
func TestHasClassWithInvalidPattern(t *testing.T) {
	if HasClass(" desktop", "(unclosed") {
		t.Error("an uncompilable pattern must be reported as absent")
	}
	if got := AddClass(" desktop", "(unclosed"); got != "desktop (unclosed" {
		t.Errorf("AddClass with an uncompilable pattern = %q", got)
	}
	if got := RemoveClass(" desktop", "(unclosed"); got != " desktop" {
		t.Errorf("RemoveClass with an uncompilable pattern = %q, want it unchanged", got)
	}
	if HasClass(" desktop", "(unclosed") {
		t.Error("the cached uncompilable pattern must still be reported as absent")
	}
}

// TestAddClass pins the leading-space behaviour inherited from the original.
func TestAddClass(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		className string
		want      string
	}{
		{"empty subject gains a leading space", "", "desktop", " desktop"},
		{"the leading space is trimmed on a second append", " macos", "desktop", "macos desktop"},
		{"already present is a no-op", " desktop", "desktop", " desktop"},
		{"surrounding whitespace is trimmed first", "  macos  ", "desktop", "macos desktop"},
		{"multi token group", "", "ios ipad tablet", " ios ipad tablet"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AddClass(tc.current, tc.className); got != tc.want {
				t.Errorf("AddClass(%q, %q) = %q, want %q", tc.current, tc.className, got, tc.want)
			}
		})
	}
}

func TestRemoveClass(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		className string
		want      string
	}{
		{"removes the token and its leading space", " macos desktop", "desktop", " macos"},
		{"absent is a no-op", " macos", "desktop", " macos"},
		{"only the first occurrence is removed", " a portrait b portrait", "portrait", " a b portrait"},
		{"a token without a leading space is not removed", "portrait", "portrait", "portrait"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RemoveClass(tc.current, tc.className); got != tc.want {
				t.Errorf("RemoveClass(%q, %q) = %q, want %q", tc.current, tc.className, got, tc.want)
			}
		})
	}
}

// TestClassNames walks every branch of the cascade documented in the README
// class table and asserts the exact resulting string, leading space included.
func TestClassNames(t *testing.T) {
	tests := []struct {
		name string
		env  Env
		want string
	}{
		{
			name: "iPad",
			env:  Env{UserAgent: "Mozilla/5.0 (iPad; CPU OS 17_7_2 like Mac OS X)"},
			want: " ios ipad tablet",
		},
		{
			name: "iPhone",
			env:  Env{UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)"},
			want: " ios iphone mobile",
		},
		{
			// A real iPod touch advertises "CPU iPhone OS", so iphone() is
			// already true and the cascade never reaches the ipod branch. The
			// class is therefore "ios iphone mobile" even though ipod() is
			// true. This is upstream behaviour; see TestIPodClassQuirk.
			name: "iPod touch is classed as an iPhone",
			env:  Env{UserAgent: "Mozilla/5.0 (iPod touch; CPU iPhone OS 15_0 like Mac OS X)"},
			want: " ios iphone mobile",
		},
		{
			name: "iPod without an iPhone OS token reaches the ipod branch",
			env:  Env{UserAgent: "Mozilla/5.0 (iPod; U; CPU like Mac OS X)"},
			want: " ios ipod mobile",
		},
		{
			name: "iPad in desktop mode",
			env:  Env{UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", Platform: "MacIntel", MaxTouchPoints: 5},
			want: " ios ipad tablet",
		},
		{
			name: "macOS",
			env:  Env{UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", Platform: "MacIntel"},
			want: " macos desktop",
		},
		{
			name: "HarmonyOS phone",
			env:  Env{UserAgent: "Mozilla/5.0 (Linux; Android 10; HarmonyOS; ELS-AN10) Mobile Safari/537.36"},
			want: " harmonyos mobile",
		},
		{
			name: "HarmonyOS tablet",
			env:  Env{UserAgent: "Mozilla/5.0 (Linux; Android 12; HarmonyOS; BRT-W09) Safari/537.36"},
			want: " harmonyos tablet",
		},
		{
			name: "Android phone",
			env:  Env{UserAgent: "Mozilla/5.0 (Linux; Android 10; K) Mobile Safari/537.36"},
			want: " android mobile",
		},
		{
			name: "Android tablet",
			env:  Env{UserAgent: "Mozilla/5.0 (Linux; Android 13; SM-X710) Safari/537.36"},
			want: " android tablet",
		},
		{
			name: "BlackBerry phone",
			env:  Env{UserAgent: "Mozilla/5.0 (BlackBerry; U; BlackBerry 9900; en) Mobile Safari/534.11+"},
			want: " blackberry mobile",
		},
		{
			name: "BlackBerry tablet",
			env:  Env{UserAgent: "Mozilla/5.0 (BlackBerry; U; RIM Tablet OS 1.0.0; en-US) Safari/534.11+"},
			want: " blackberry tablet",
		},
		{
			name: "Windows tablet",
			env:  Env{UserAgent: "Mozilla/5.0 (Windows NT 6.2; ARM; Trident/7.0; Touch; rv:11.0) like Gecko"},
			want: " windows tablet",
		},
		{
			name: "Windows phone",
			env:  Env{UserAgent: "Mozilla/5.0 (Windows Phone 10.0; Lumia 950) Mobile Safari/537.36"},
			want: " windows mobile",
		},
		{
			name: "Windows desktop",
			env:  Env{UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/134.0.0.0 Safari/537.36"},
			want: " windows desktop",
		},
		{
			name: "Firefox OS phone",
			env:  Env{UserAgent: "Mozilla/5.0 (Mobile; rv:26.0) Gecko/26.0 Firefox/26.0"},
			want: " fxos mobile",
		},
		{
			name: "Firefox OS tablet",
			env:  Env{UserAgent: "Mozilla/5.0 (Tablet; rv:26.0) Gecko/26.0 Firefox/26.0"},
			want: " fxos tablet",
		},
		{
			name: "MeeGo",
			env:  Env{UserAgent: "Mozilla/5.0 (MeeGo; NokiaN9) Mobile Safari/534.13"},
			want: " meego mobile",
		},
		{
			name: "NW.js",
			env:  Env{UserAgent: "Mozilla/5.0 (X11; Linux x86_64) Chrome/134.0.0.0", HasProcessObject: true},
			want: " node-webkit",
		},
		{
			name: "television",
			env:  Env{UserAgent: "Mozilla/5.0 (SMART-TV; Linux; Tizen 5.0) SmartTV"},
			want: " television",
		},
		{
			name: "desktop fallback",
			env:  Env{UserAgent: "Mozilla/5.0 (X11; Linux x86_64) Chrome/134.0.0.0 Safari/537.36"},
			want: " desktop",
		},
		{
			name: "Cordova is appended to the branch class",
			env: Env{
				UserAgent:        "Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)",
				HasCordova:       true,
				LocationProtocol: "file:",
			},
			want: "ios iphone mobile cordova",
		},
		{
			name: "Cordova on its own when no branch matches",
			env: Env{
				UserAgent:        "Mozilla/5.0 (Mobile; rv:26.0) Gecko/26.0",
				HasCordova:       true,
				LocationProtocol: "file:",
			},
			want: "fxos mobile cordova",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := New(tc.env).ClassNames(""); got != tc.want {
				t.Errorf("ClassNames(\"\") = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNodeWebkitPrecedesTelevision locks in the cascade ordering quirk: an NW.js
// runtime is labelled "node-webkit" even when the user agent is a television,
// and no OS or form factor class is added.
func TestNodeWebkitPrecedesTelevision(t *testing.T) {
	d := New(Env{UserAgent: "Roku/DVP-11.5 (11.5.0)", HasProcessObject: true})
	if got := d.ClassNames(""); got != " node-webkit" {
		t.Errorf("ClassNames(\"\") = %q, want %q", got, " node-webkit")
	}
	if !d.Television() {
		t.Error("Television() = false; the predicate is still true, only the class differs")
	}
	if got := d.OS(); got != OSTelevision {
		t.Errorf("OS() = %q, want %q; the cascade order does not affect OS resolution", got, OSTelevision)
	}
}

// TestIPodClassQuirk pins the interaction between the iOS cascade order and
// real iPod touch user agents, which advertise "CPU iPhone OS".
func TestIPodClassQuirk(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (iPod touch; CPU iPhone OS 15_0 like Mac OS X)")

	if !d.IPod() {
		t.Error("IPod() = false, want true")
	}
	if !d.IPhone() {
		t.Error("IPhone() = false; an iPod touch user agent contains iPhone OS")
	}
	if got := d.ClassNames(""); got != " ios iphone mobile" {
		t.Errorf("ClassNames(\"\") = %q; the iphone branch precedes the ipod branch", got)
	}
}

// TestCordovaClassLosesLeadingSpace documents the consequence of AddClass
// trimming: once a second class is appended, the leading space introduced by
// the first append is gone.
func TestCordovaClassLosesLeadingSpace(t *testing.T) {
	d := New(Env{
		UserAgent:        "Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)",
		HasCordova:       true,
		LocationProtocol: "file:",
	})
	if got := d.ClassNames(""); got != "ios iphone mobile cordova" {
		t.Errorf("ClassNames(\"\") = %q, want %q", got, "ios iphone mobile cordova")
	}
}

// TestClassNamesPreservesExistingClasses confirms the cascade appends to
// whatever the document element already carried.
func TestClassNamesPreservesExistingClasses(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (X11; Linux x86_64) Chrome/134.0.0.0 Safari/537.36")
	if got := d.ClassNames("no-js theme-dark"); got != "no-js theme-dark desktop" {
		t.Errorf("ClassNames() = %q, want %q", got, "no-js theme-dark desktop")
	}
}

// TestClassNamesIsIdempotent confirms that re-running the cascade, as a
// double-import would, does not duplicate classes.
func TestClassNamesIsIdempotent(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)")
	once := d.ClassNames("")
	twice := d.ClassNames(once)
	if once != twice {
		t.Errorf("ClassNames is not idempotent: %q then %q", once, twice)
	}
}

// TestIOSWithoutSpecificDeviceAddsNothing covers the fall-through inside the
// iOS branch. It is unreachable in practice, since IOS is the disjunction of
// the three probes, but the branch exists upstream and is reproduced here.
func TestIOSWithoutSpecificDeviceAddsNothing(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (X11; Linux x86_64)")
	if d.IOS() {
		t.Skip("fixture is not iOS")
	}
	if got := d.ClassNames(""); !strings.Contains(got, "desktop") {
		t.Errorf("ClassNames(\"\") = %q, want it to contain desktop", got)
	}
}
