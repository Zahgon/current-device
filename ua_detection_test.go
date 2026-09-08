package device

import (
	"fmt"
	"sort"
	"testing"
)

// TestUAStringDetection is the one-for-one port of tests/ua-detection.test.ts.
// The subtest names reproduce the source describe/it strings verbatim
// ("<fixture> > detects os=..., type=..." and "<method>() returns <bool>") so
// each TypeScript test has a machine-matchable counterpart here.
func TestUAStringDetection(t *testing.T) {
	for _, fixture := range uaFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			d := New(fixture.env())

			t.Run(fmt.Sprintf("detects os=%s, type=%s", fixture.expectedOS, fixture.expectedType), func(t *testing.T) {
				if got := d.OS(); got != fixture.expectedOS {
					t.Errorf("OS() = %q, want %q", got, fixture.expectedOS)
				}
				if got := d.Type(); got != fixture.expectedType {
					t.Errorf("Type() = %q, want %q", got, fixture.expectedType)
				}
			})

			// Iterate deterministically so failures are reproducible.
			names := make([]string, 0, len(fixture.methods))
			for name := range fixture.methods {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				want := fixture.methods[name]
				t.Run(fmt.Sprintf("%s() returns %t", name, want), func(t *testing.T) {
					fn, ok := d.Predicate(name)
					if !ok {
						t.Fatalf("Predicate(%q) is not registered", name)
					}
					if got := fn(); got != want {
						t.Errorf("%s() = %v, want %v", name, got, want)
					}
				})
			}
		})
	}
}

// TestFixtureCoverage guards against a fixture being dropped during future
// edits: tests/ua-strings.ts ships exactly 23 of them.
func TestFixtureCoverage(t *testing.T) {
	const wantFixtures = 23
	if len(uaFixtures) != wantFixtures {
		t.Errorf("len(uaFixtures) = %d, want %d", len(uaFixtures), wantFixtures)
	}

	seen := make(map[string]bool, len(uaFixtures))
	for _, f := range uaFixtures {
		if seen[f.name] {
			t.Errorf("duplicate fixture name %q", f.name)
		}
		seen[f.name] = true
	}
}

// TestResolvedValuesAreAlwaysDeclaredConstants asserts that no user agent can
// drive Type, OS or Orientation outside their declared value sets.
func TestResolvedValuesAreAlwaysDeclaredConstants(t *testing.T) {
	validTypes := map[DeviceType]bool{
		TypeMobile: true, TypeTablet: true, TypeDesktop: true, TypeUnknown: true,
	}
	validOS := map[DeviceOS]bool{
		OSIOS: true, OSIPhone: true, OSIPad: true, OSIPod: true, OSAndroid: true,
		OSBlackberry: true, OSMacOS: true, OSWindows: true, OSFxOS: true,
		OSMeeGo: true, OSTelevision: true, OSHarmonyOS: true, OSUnknown: true,
	}
	validOrientation := map[DeviceOrientation]bool{
		OrientationLandscape: true, OrientationPortrait: true, OrientationUnknown: true,
	}

	uas := []string{"", " ", "?", "\x00\xff", "mac ipad windows android harmonyos meego roku"}
	for _, f := range uaFixtures {
		uas = append(uas, f.ua)
	}

	for _, ua := range uas {
		d := NewFromUserAgent(ua)
		if !validTypes[d.Type()] {
			t.Errorf("ua %q produced undeclared Type %q", ua, d.Type())
		}
		if !validOS[d.OS()] {
			t.Errorf("ua %q produced undeclared OS %q", ua, d.OS())
		}
		if !validOrientation[d.Orientation()] {
			t.Errorf("ua %q produced undeclared Orientation %q", ua, d.Orientation())
		}
	}
}

// FuzzDetectionNeverPanics mirrors the parity fuzz requirement: arbitrary user
// agent bytes must never panic and must always resolve to declared constants.
func FuzzDetectionNeverPanics(f *testing.F) {
	for _, fixture := range uaFixtures {
		f.Add(fixture.ua)
	}
	f.Add("")
	f.Add("(mobile rv:")

	f.Fuzz(func(t *testing.T, ua string) {
		d := New(Env{UserAgent: ua, Platform: "MacIntel", MaxTouchPoints: 5})

		switch d.Type() {
		case TypeMobile, TypeTablet, TypeDesktop, TypeUnknown:
		default:
			t.Fatalf("undeclared Type %q for ua %q", d.Type(), ua)
		}

		switch d.OS() {
		case OSIOS, OSIPhone, OSIPad, OSIPod, OSAndroid, OSBlackberry, OSMacOS,
			OSWindows, OSFxOS, OSMeeGo, OSTelevision, OSHarmonyOS, OSUnknown:
		default:
			t.Fatalf("undeclared OS %q for ua %q", d.OS(), ua)
		}

		// The class cascade must also survive arbitrary input.
		_ = d.ClassNames("")
		_ = d.HandleOrientation("")
	})
}
