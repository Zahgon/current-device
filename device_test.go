package device

import (
	"math"
	"testing"
)

// TestPredicateRegistryIsConsistent guards the name-based dispatch that the
// ported suites rely on: PredicateNames and the registry must agree exactly,
// and the count must stay at the 27 the TypeScript object exposed.
func TestPredicateRegistryIsConsistent(t *testing.T) {
	names := PredicateNames()
	if len(names) != 27 {
		t.Fatalf("PredicateNames() returned %d names, want 27", len(names))
	}
	if len(predicates) != len(names) {
		t.Errorf("registry holds %d predicates but PredicateNames lists %d", len(predicates), len(names))
	}
	for _, name := range names {
		if _, ok := predicates[name]; !ok {
			t.Errorf("PredicateNames lists %q which is absent from the registry", name)
		}
	}
}

// TestBlackberry covers a family the UA fixtures never exercise.
func TestBlackberry(t *testing.T) {
	tests := []struct {
		name                              string
		ua                                string
		blackberry, phone, tablet, mobile bool
	}{
		{
			name:       "BlackBerry OS 7 phone",
			ua:         "Mozilla/5.0 (BlackBerry; U; BlackBerry 9900; en) AppleWebKit/534.11+ (KHTML, like Gecko) Version/7.1.0.346 Mobile Safari/534.11+",
			blackberry: true, phone: true, tablet: false, mobile: true,
		},
		{
			name:       "BlackBerry 10 phone",
			ua:         "Mozilla/5.0 (BB10; Touch) AppleWebKit/537.10+ (KHTML, like Gecko) Version/10.0.9.2372 Mobile Safari/537.10+",
			blackberry: true, phone: true, tablet: false, mobile: true,
		},
		{
			name:       "BlackBerry PlayBook tablet",
			ua:         "Mozilla/5.0 (PlayBook; U; RIM Tablet OS 2.1.0; en-US) AppleWebKit/536.2+ (KHTML like Gecko) Version/7.2.1.0 Safari/536.2+",
			blackberry: false, phone: false, tablet: false, mobile: false,
		},
		{
			name:       "BlackBerry-branded tablet",
			ua:         "Mozilla/5.0 (BlackBerry; U; RIM Tablet OS 1.0.0; en-US) AppleWebKit/534.11+ (KHTML, like Gecko) Version/7.1.0.7 Safari/534.11+",
			blackberry: true, phone: false, tablet: true, mobile: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewFromUserAgent(tc.ua)
			if got := d.Blackberry(); got != tc.blackberry {
				t.Errorf("Blackberry() = %v, want %v", got, tc.blackberry)
			}
			if got := d.BlackberryPhone(); got != tc.phone {
				t.Errorf("BlackberryPhone() = %v, want %v", got, tc.phone)
			}
			if got := d.BlackberryTablet(); got != tc.tablet {
				t.Errorf("BlackberryTablet() = %v, want %v", got, tc.tablet)
			}
			if got := d.Mobile(); got != tc.mobile {
				t.Errorf("Mobile() = %v, want %v", got, tc.mobile)
			}
		})
	}
}

// TestBlackberryOSAndType pins the resolved labels for a BlackBerry phone.
func TestBlackberryOSAndType(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (BB10; Touch) AppleWebKit/537.10+ (KHTML, like Gecko) Version/10.0.9.2372 Mobile Safari/537.10+")
	if got := d.OS(); got != OSBlackberry {
		t.Errorf("OS() = %q, want %q", got, OSBlackberry)
	}
	if got := d.Type(); got != TypeMobile {
		t.Errorf("Type() = %q, want %q", got, TypeMobile)
	}
}

// TestFxOS covers Firefox OS, which no UA fixture exercises. The negative cases
// are the important ones: desktop and mobile Firefox user agents also contain
// "rv:", and only the exact "(mobile"/"(tablet" plus " rv:" needles keep them
// out.
func TestFxOS(t *testing.T) {
	tests := []struct {
		name                string
		ua                  string
		fxos, phone, tablet bool
		mobileType          bool
		expectedOSIsFirefox bool
	}{
		{
			name: "Firefox OS phone",
			ua:   "Mozilla/5.0 (Mobile; rv:26.0) Gecko/26.0 Firefox/26.0",
			fxos: true, phone: true, tablet: false, mobileType: true, expectedOSIsFirefox: true,
		},
		{
			name: "Firefox OS tablet",
			ua:   "Mozilla/5.0 (Tablet; rv:26.0) Gecko/26.0 Firefox/26.0",
			fxos: true, phone: false, tablet: true, mobileType: false, expectedOSIsFirefox: true,
		},
		{
			name: "desktop Firefox is not Firefox OS",
			ua:   "Mozilla/5.0 (X11; Linux x86_64; rv:135.0) Gecko/20100101 Firefox/135.0",
			fxos: false, phone: false, tablet: false, mobileType: false, expectedOSIsFirefox: false,
		},
		{
			// "; mobile" is not "(mobile", so the leading parenthesis in the
			// needle is what keeps Firefox for Android out of the Firefox OS
			// branch and lets it resolve as Android instead.
			name: "Firefox for Android is Android, not Firefox OS",
			ua:   "Mozilla/5.0 (Android 13; Mobile; rv:135.0) Gecko/135.0 Firefox/135.0",
			fxos: false, phone: false, tablet: false, mobileType: true, expectedOSIsFirefox: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewFromUserAgent(tc.ua)
			if got := d.FxOS(); got != tc.fxos {
				t.Errorf("FxOS() = %v, want %v", got, tc.fxos)
			}
			if got := d.FxOSPhone(); got != tc.phone {
				t.Errorf("FxOSPhone() = %v, want %v", got, tc.phone)
			}
			if got := d.FxOSTablet(); got != tc.tablet {
				t.Errorf("FxOSTablet() = %v, want %v", got, tc.tablet)
			}
			if got := d.Mobile(); got != tc.mobileType {
				t.Errorf("Mobile() = %v, want %v", got, tc.mobileType)
			}
			if got := d.OS() == OSFxOS; got != tc.expectedOSIsFirefox {
				t.Errorf("OS() == fxos is %v, want %v (OS is %q)", got, tc.expectedOSIsFirefox, d.OS())
			}
		})
	}
}

// TestMeeGo covers MeeGo, which no UA fixture exercises.
func TestMeeGo(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (MeeGo; NokiaN9) AppleWebKit/534.13 (KHTML, like Gecko) NokiaBrowser/8.5.0 Mobile Safari/534.13")
	if !d.MeeGo() {
		t.Error("MeeGo() = false, want true")
	}
	if !d.Mobile() {
		t.Error("Mobile() = false, want true")
	}
	if got := d.OS(); got != OSMeeGo {
		t.Errorf("OS() = %q, want %q", got, OSMeeGo)
	}
	if got := d.Type(); got != TypeMobile {
		t.Errorf("Type() = %q, want %q", got, TypeMobile)
	}
}

// TestWindowsTablet covers the touch-enabled Windows branch, which no UA
// fixture exercises, together with the Windows Phone exclusion that guards it.
func TestWindowsTablet(t *testing.T) {
	tablet := NewFromUserAgent("Mozilla/5.0 (Windows NT 6.2; ARM; Trident/7.0; Touch; rv:11.0; WPDesktop) like Gecko")
	if !tablet.WindowsTablet() {
		t.Error("WindowsTablet() = false, want true")
	}
	if !tablet.Tablet() {
		t.Error("Tablet() = false, want true")
	}
	if got := tablet.Type(); got != TypeTablet {
		t.Errorf("Type() = %q, want %q", got, TypeTablet)
	}

	phone := NewFromUserAgent("Mozilla/5.0 (Windows Phone 10.0; Touch; Lumia 950)")
	if phone.WindowsTablet() {
		t.Error("a touch-enabled Windows Phone must not be a WindowsTablet")
	}
	if !phone.WindowsPhone() {
		t.Error("WindowsPhone() = false, want true")
	}
}

// TestTelevisionList asserts every entry of the television needle list is
// detected, since only three of the fourteen appear in the UA fixtures.
func TestTelevisionList(t *testing.T) {
	for _, needle := range televisionDevices {
		t.Run(needle, func(t *testing.T) {
			d := NewFromUserAgent("Mozilla/5.0 (" + needle + ")")
			if !d.Television() {
				t.Errorf("Television() = false for needle %q", needle)
			}
			if got := d.OS(); got != OSTelevision {
				t.Errorf("OS() = %q, want %q", got, OSTelevision)
			}
			if got := d.Type(); got != TypeDesktop {
				t.Errorf("Type() = %q, want %q; televisions fall through to desktop", got, TypeDesktop)
			}
		})
	}

	if got := len(televisionDevices); got != 14 {
		t.Errorf("televisionDevices has %d entries, want 14", got)
	}

	if NewFromUserAgent("Mozilla/5.0 (X11; Linux x86_64)").Television() {
		t.Error("Television() = true for a plain Linux user agent")
	}
}

// TestCordova exercises the runtime probe, which depends on Env rather than the
// user agent.
func TestCordova(t *testing.T) {
	tests := []struct {
		name       string
		hasCordova bool
		protocol   string
		want       bool
	}{
		{"cordova object over file protocol", true, "file:", true},
		{"cordova object over https", true, "https:", false},
		{"no cordova object over file protocol", false, "file:", false},
		{"neither", false, "https:", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New(Env{
				UserAgent:        "Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)",
				HasCordova:       tc.hasCordova,
				LocationProtocol: tc.protocol,
			})
			if got := d.Cordova(); got != tc.want {
				t.Errorf("Cordova() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNodeWebkit exercises the NW.js probe.
func TestNodeWebkit(t *testing.T) {
	if !New(Env{UserAgent: "Mozilla/5.0", HasProcessObject: true}).NodeWebkit() {
		t.Error("NodeWebkit() = false when a process object is present")
	}
	if New(Env{UserAgent: "Mozilla/5.0"}).NodeWebkit() {
		t.Error("NodeWebkit() = true when no process object is present")
	}
}

// TestIPadDesktopModeDisambiguation pins the navigator.platform and
// navigator.maxTouchPoints heuristic, including the boundary at exactly one
// touch point.
func TestIPadDesktopModeDisambiguation(t *testing.T) {
	const macUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.10 Safari/605.1.15"

	tests := []struct {
		name           string
		platform       string
		maxTouchPoints int
		wantIPad       bool
		wantMacOS      bool
	}{
		{"iPadOS 13+ desktop mode", "MacIntel", 5, true, false},
		{"real Mac", "MacIntel", 0, false, true},
		{"Mac with exactly one touch point", "MacIntel", 1, false, true},
		{"non-MacIntel platform with touch", "Win32", 5, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New(Env{UserAgent: macUA, Platform: tc.platform, MaxTouchPoints: tc.maxTouchPoints})
			if got := d.IPad(); got != tc.wantIPad {
				t.Errorf("IPad() = %v, want %v", got, tc.wantIPad)
			}
			if got := d.MacOS(); got != tc.wantMacOS {
				t.Errorf("MacOS() = %v, want %v", got, tc.wantMacOS)
			}
		})
	}
}

// TestPlatformIsCaseSensitive documents that navigator.platform is compared
// exactly, unlike the user agent which is lower-cased.
func TestPlatformIsCaseSensitive(t *testing.T) {
	d := New(Env{UserAgent: "Mozilla/5.0 (Macintosh)", Platform: "macintel", MaxTouchPoints: 5})
	if d.IPad() {
		t.Error(`IPad() = true for platform "macintel"; the comparison against "MacIntel" is case sensitive`)
	}
}

// TestUserAgentIsLowercased confirms the single normalisation performed by New,
// which is what allows every probe to use plain substring search.
func TestUserAgentIsLowercased(t *testing.T) {
	d := NewFromUserAgent("MOZILLA/5.0 (IPHONE; CPU IPHONE OS 18_3_2 LIKE MAC OS X)")
	if !d.IPhone() {
		t.Error("IPhone() = false for an upper-cased user agent")
	}
	if got := d.Env().UserAgent; got != "mozilla/5.0 (iphone; cpu iphone os 18_3_2 like mac os x)" {
		t.Errorf("Env().UserAgent = %q, want it lower-cased", got)
	}
}

// TestIOSShadowsSpecificOSLabels locks in the findMatch ordering consequence
// that every iOS device reports OS "ios" and never "iphone", "ipad" or "ipod".
func TestIOSShadowsSpecificOSLabels(t *testing.T) {
	for _, ua := range []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)",
		"Mozilla/5.0 (iPad; CPU OS 17_7_2 like Mac OS X)",
		"Mozilla/5.0 (iPod touch; CPU iPhone OS 15_0 like Mac OS X)",
	} {
		if got := NewFromUserAgent(ua).OS(); got != OSIOS {
			t.Errorf("OS() = %q for %q, want %q", got, ua, OSIOS)
		}
	}
}

// TestWindowsSuppressesIPhoneAndAndroid documents the !Windows() guards, which
// stop a Windows Phone user agent that also advertises Android from being
// detected as Android.
func TestWindowsSuppressesIPhoneAndAndroid(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (Windows Phone 10.0; Android 6.0.1; Microsoft; Lumia 950) Mobile Safari/537.36")
	if d.Android() {
		t.Error("Android() = true; the Windows guard should suppress it")
	}
	if got := d.OS(); got != OSWindows {
		t.Errorf("OS() = %q, want %q", got, OSWindows)
	}
}

// TestHarmonyOSPrecedesAndroid locks in the findMatch ordering that resolves
// HarmonyOS user agents, which also contain "android", to "harmonyos".
func TestHarmonyOSPrecedesAndroid(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (Linux; Android 10; HarmonyOS; ELS-AN10) Mobile Safari/537.36")
	if !d.Android() {
		t.Error("Android() = false; a HarmonyOS user agent still contains android")
	}
	if got := d.OS(); got != OSHarmonyOS {
		t.Errorf("OS() = %q, want %q", got, OSHarmonyOS)
	}
}

// TestUnknownPredicateName covers the negative branch of the dynamic dispatch.
func TestUnknownPredicateName(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0")
	if fn, ok := d.Predicate("nosuchmethod"); ok || fn != nil {
		t.Error("Predicate() must report unknown names")
	}
	if got := d.findMatch([]string{"nosuchmethod"}); got != "unknown" {
		t.Errorf("findMatch with an unknown name = %q, want %q", got, "unknown")
	}
}

// TestSetEnvDoesNotRecomputeCaches documents that Type and OS are resolved once
// at construction, exactly as the TypeScript module resolved them once at
// import, while orientation is refreshed by HandleOrientation.
func TestSetEnvDoesNotRecomputeCaches(t *testing.T) {
	d := NewFromUserAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X)")
	if got := d.OS(); got != OSIOS {
		t.Fatalf("OS() = %q, want %q", got, OSIOS)
	}

	d.SetEnv(Env{UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"})

	if got := d.OS(); got != OSIOS {
		t.Errorf("OS() = %q after SetEnv, want the cached %q", got, OSIOS)
	}
	if !d.Windows() {
		t.Error("Windows() = false after SetEnv; live probes must read the new environment")
	}
	if got := d.Env().UserAgent; got != "mozilla/5.0 (windows nt 10.0; win64; x64)" {
		t.Errorf("Env().UserAgent = %q, want the new lower-cased agent", got)
	}
}

// TestZeroViewportMatchesJavaScriptSemantics documents that a zero-sized
// viewport yields IEEE-754 infinities and NaNs rather than a panic, matching
// JavaScript division semantics.
func TestZeroViewportMatchesJavaScriptSemantics(t *testing.T) {
	t.Run("zero width with positive height is portrait", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 0, InnerHeight: 800})
		width, height := d.Env().InnerWidth, d.Env().InnerHeight
		if !math.IsInf(height/width, 1) {
			t.Fatal("expected positive infinity from division by zero")
		}
		if !d.Portrait() {
			t.Error("Portrait() = false; Infinity > 1 is true")
		}
		if d.Landscape() {
			t.Error("Landscape() = true; Infinity < 1 is false")
		}
		if got := d.Orientation(); got != OrientationPortrait {
			t.Errorf("Orientation() = %q, want %q", got, OrientationPortrait)
		}
	})

	t.Run("fully zero viewport is unknown", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0"})
		if d.Portrait() {
			t.Error("Portrait() = true; every NaN comparison is false")
		}
		if d.Landscape() {
			t.Error("Landscape() = true; every NaN comparison is false")
		}
		if got := d.Orientation(); got != OrientationUnknown {
			t.Errorf("Orientation() = %q, want %q", got, OrientationUnknown)
		}
	})
}
