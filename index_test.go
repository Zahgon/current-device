package device

import (
	"fmt"
	"testing"
)

// TestCurrentDevice is the one-for-one port of tests/index.test.ts. Subtest
// names reproduce the source describe/it strings verbatim so each TypeScript
// test has a machine-matchable counterpart here.
func TestCurrentDevice(t *testing.T) {
	// The TypeScript suite runs under jsdom, whose navigator reports a Linux
	// desktop UA. The same UA is used here so the ported assertions observe the
	// same device.
	const jsdomUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"

	newDevice := func() *Device {
		return New(Env{
			UserAgent:   jsdomUA,
			InnerWidth:  1024,
			InnerHeight: 768,
		})
	}

	t.Run("Exports an `object`", func(t *testing.T) {
		if d := newDevice(); d == nil {
			t.Fatal("New returned nil, want a *Device value")
		}
	})

	t.Run("Exposes attributes for os, type, and orientation", func(t *testing.T) {
		d := newDevice()

		t.Run("Exposes `os` string", func(t *testing.T) {
			if d.OS() == "" {
				t.Error("OS() is empty, want a non-empty string")
			}
		})
		t.Run("Exposes `type` string", func(t *testing.T) {
			if d.Type() == "" {
				t.Error("Type() is empty, want a non-empty string")
			}
		})
		t.Run("Exposes `orientation` string", func(t *testing.T) {
			if d.Orientation() == "" {
				t.Error("Orientation() is empty, want a non-empty string")
			}
		})
	})

	// exposesFunction mirrors the source assertion `expect(typeof
	// device[name]).toBe('function')`: the predicate must be registered under
	// its TypeScript name and must be callable, returning a bool.
	exposesFunction := func(t *testing.T, d *Device, name string) {
		t.Helper()
		fn, ok := d.Predicate(name)
		if !ok {
			t.Fatalf("Predicate(%q) is not registered", name)
		}
		if fn == nil {
			t.Fatalf("Predicate(%q) returned a nil function", name)
		}
		_ = fn()
	}

	runGroup := func(t *testing.T, group string, names []string) {
		t.Helper()
		t.Run(group, func(t *testing.T) {
			d := newDevice()
			for _, name := range names {
				t.Run(fmt.Sprintf("Exposes a `%s` function", name), func(t *testing.T) {
					exposesFunction(t, d, name)
				})
			}
		})
	}

	t.Run("Exposes functions for detecting device `os`", func(t *testing.T) {
		runGroup(t, "Apple (iOS, macOS)", []string{"macos", "ios", "iphone", "ipad", "ipod"})
		runGroup(t, "Android", []string{"android", "androidPhone", "androidTablet"})
		runGroup(t, "Blackberry", []string{"blackberry", "blackberryPhone", "blackberryTablet"})
		runGroup(t, "Windows", []string{"windows", "windowsPhone", "windowsTablet"})
		runGroup(t, "Firefox OS", []string{"fxos", "fxosPhone", "fxosTablet"})
		runGroup(t, "Other", []string{"meego", "harmonyos", "cordova", "nodeWebkit"})
	})

	t.Run("Exposes functions for detecting device `type`", func(t *testing.T) {
		d := newDevice()
		for _, name := range []string{"desktop", "tablet", "mobile", "television"} {
			t.Run(fmt.Sprintf("Exposes a `%s` function", name), func(t *testing.T) {
				exposesFunction(t, d, name)
			})
		}
	})

	t.Run("Exposes functions for detecting device `orientation`", func(t *testing.T) {
		d := newDevice()
		for _, name := range []string{"portrait", "landscape"} {
			t.Run(fmt.Sprintf("Exposes a `%s` function", name), func(t *testing.T) {
				exposesFunction(t, d, name)
			})
		}
	})

	t.Run("Exposes helper functions", func(t *testing.T) {
		t.Run("Exposes a `noConflict` function", func(t *testing.T) {
			if got := newDevice().NoConflict(); got == nil {
				t.Fatal("NoConflict() returned nil, want the receiver")
			}
		})

		// The source asserts that `noConflict()` puts the prior window.device
		// back. Restoring a browser global is only expressible in the syscall/js
		// layer, where cmd/wasm's TestNoConflictRestoresPreviousGlobal and
		// TestNoConflictRestoresUndefined pin exactly that. What the core package
		// can and must guarantee is the other half of the contract the source
		// relies on: the call returns the same device so `var d = device.noConflict()`
		// keeps working.
		t.Run("Restores the previous value of the `device` global object when `noConflict` is called", func(t *testing.T) {
			d := newDevice()
			if got := d.NoConflict(); got != d {
				t.Errorf("NoConflict() = %p, want the receiver %p", got, d)
			}
		})

		t.Run("Exposes a `onChangeOrientation` function", func(t *testing.T) {
			d := newDevice()
			d.OnChangeOrientation(func(DeviceOrientation) {})
		})

		t.Run("Calls the provided callback when orientation changes using `onChangeOrientation`", func(t *testing.T) {
			d := New(Env{UserAgent: jsdomUA, InnerWidth: 1024, InnerHeight: 768})

			var got []DeviceOrientation
			d.OnChangeOrientation(func(o DeviceOrientation) {
				got = append(got, o)
			})

			// A landscape viewport resolves landscape; flipping the viewport and
			// re-running the handler must report portrait.
			d.HandleOrientation("")
			d.SetEnv(Env{UserAgent: jsdomUA, InnerWidth: 768, InnerHeight: 1024})
			d.HandleOrientation("")

			want := []DeviceOrientation{OrientationLandscape, OrientationPortrait}
			if len(got) != len(want) {
				t.Fatalf("callback fired %d time(s) with %v, want %d: %v", len(got), got, len(want), want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("callback %d received %q, want %q", i, got[i], want[i])
				}
			}
		})
	})

	t.Run("HTML Element Handling", func(t *testing.T) {
		// ClassNames is the pure function that computes what the source writes to
		// documentElement.className; cmd/wasm's TestInstallStampsDetectionClasses
		// covers the DOM write itself.
		t.Run("Adds the correct CSS classes to the <html> element based on the user agent", func(t *testing.T) {
			tests := []struct {
				ua   string
				want string
			}{
				{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15", " ios iphone mobile"},
				{"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15", " ios ipad tablet"},
				{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15", " macos desktop"},
				{"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Mobile Safari/537.36", " android mobile"},
				{"Mozilla/5.0 (Linux; Android 14; SM-X710) AppleWebKit/537.36 Safari/537.36", " android tablet"},
				{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/134.0.0.0", " windows desktop"},
				{jsdomUA, " desktop"},
			}

			for _, tc := range tests {
				d := NewFromUserAgent(tc.ua)
				if got := d.ClassNames(""); got != tc.want {
					t.Errorf("ClassNames(\"\") for %q = %q, want %q", tc.ua, got, tc.want)
				}
			}
		})
	})
}
