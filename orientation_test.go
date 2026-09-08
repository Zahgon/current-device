package device

import (
	"sync"
	"testing"
)

const iPhoneUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1"

// TestOrientationScreenOrientationAPI covers the first strategy, which requires
// both a screen.orientation object and window.onorientationchange.
func TestOrientationScreenOrientationAPI(t *testing.T) {
	tests := []struct {
		name                                   string
		orientationType                        string
		hasScreenOrientation, hasOnOrientation bool
		wantPortrait, wantLandscape            bool
		wantOrientation                        DeviceOrientation
	}{
		{
			name: "portrait-primary", orientationType: "portrait-primary",
			hasScreenOrientation: true, hasOnOrientation: true,
			wantPortrait: true, wantLandscape: false, wantOrientation: OrientationPortrait,
		},
		{
			name: "portrait-secondary", orientationType: "portrait-secondary",
			hasScreenOrientation: true, hasOnOrientation: true,
			wantPortrait: true, wantLandscape: false, wantOrientation: OrientationPortrait,
		},
		{
			name: "landscape-primary", orientationType: "landscape-primary",
			hasScreenOrientation: true, hasOnOrientation: true,
			wantPortrait: false, wantLandscape: true, wantOrientation: OrientationLandscape,
		},
		{
			name: "landscape-secondary", orientationType: "landscape-secondary",
			hasScreenOrientation: true, hasOnOrientation: true,
			wantPortrait: false, wantLandscape: true, wantOrientation: OrientationLandscape,
		},
		{
			name: "unrecognised type falls through to unknown", orientationType: "square",
			hasScreenOrientation: true, hasOnOrientation: true,
			wantPortrait: false, wantLandscape: false, wantOrientation: OrientationUnknown,
		},
		{
			// Without onorientationchange the API branch is skipped entirely,
			// so the 1024x768 viewport below decides instead.
			name: "screen.orientation alone is not enough", orientationType: "portrait-primary",
			hasScreenOrientation: true, hasOnOrientation: false,
			wantPortrait: false, wantLandscape: true, wantOrientation: OrientationLandscape,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New(Env{
				UserAgent:              "Mozilla/5.0 (X11; Linux x86_64)",
				HasScreenOrientation:   tc.hasScreenOrientation,
				HasOnOrientationChange: tc.hasOnOrientation,
				ScreenOrientationType:  tc.orientationType,
				InnerWidth:             1024,
				InnerHeight:            768,
			})

			if got := d.Portrait(); got != tc.wantPortrait {
				t.Errorf("Portrait() = %v, want %v", got, tc.wantPortrait)
			}
			if got := d.Landscape(); got != tc.wantLandscape {
				t.Errorf("Landscape() = %v, want %v", got, tc.wantLandscape)
			}
			if got := d.Orientation(); got != tc.wantOrientation {
				t.Errorf("Orientation() = %q, want %q", got, tc.wantOrientation)
			}
		})
	}
}

// TestOrientationLegacyIOSAngle covers the second strategy, the legacy
// window.orientation angle, which only applies on iOS.
func TestOrientationLegacyIOSAngle(t *testing.T) {
	tests := []struct {
		name            string
		angle           int
		wantOrientation DeviceOrientation
	}{
		{"upright", 0, OrientationPortrait},
		{"rotated right", 90, OrientationLandscape},
		{"rotated left", -90, OrientationLandscape},
		{"upside down", 180, OrientationPortrait},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New(Env{
				UserAgent:            iPhoneUA,
				HasWindowOrientation: true,
				WindowOrientation:    tc.angle,
			})
			if got := d.Orientation(); got != tc.wantOrientation {
				t.Errorf("Orientation() at %d degrees = %q, want %q", tc.angle, got, tc.wantOrientation)
			}
		})
	}
}

// TestLegacyAngleRequiresIOS confirms the angle branch is gated on iOS, so a
// non-iOS device exposing window.orientation falls through to the ratio.
func TestLegacyAngleRequiresIOS(t *testing.T) {
	d := New(Env{
		UserAgent:            "Mozilla/5.0 (Linux; Android 10; K) Mobile Safari/537.36",
		HasWindowOrientation: true,
		WindowOrientation:    90,
		InnerWidth:           400,
		InnerHeight:          800,
	})
	if got := d.Orientation(); got != OrientationPortrait {
		t.Errorf("Orientation() = %q, want %q from the viewport ratio", got, OrientationPortrait)
	}
}

// TestOrientationViewportRatio covers the third strategy and the square
// viewport quirk it produces.
func TestOrientationViewportRatio(t *testing.T) {
	tests := []struct {
		name            string
		width, height   float64
		wantOrientation DeviceOrientation
	}{
		{"taller than wide", 400, 800, OrientationPortrait},
		{"wider than tall", 1024, 768, OrientationLandscape},
		{"exactly square is unknown", 500, 500, OrientationUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New(Env{UserAgent: "Mozilla/5.0 (X11; Linux x86_64)", InnerWidth: tc.width, InnerHeight: tc.height})
			if got := d.Orientation(); got != tc.wantOrientation {
				t.Errorf("Orientation() for %vx%v = %q, want %q", tc.width, tc.height, got, tc.wantOrientation)
			}
		})
	}
}

// TestSquareViewportBothProbesFalse states the quirk explicitly: Portrait and
// Landscape are independent probes, not complements.
func TestSquareViewportBothProbesFalse(t *testing.T) {
	d := New(Env{UserAgent: "Mozilla/5.0 (X11; Linux x86_64)", InnerWidth: 500, InnerHeight: 500})
	if d.Portrait() {
		t.Error("Portrait() = true on a square viewport")
	}
	if d.Landscape() {
		t.Error("Landscape() = true on a square viewport")
	}
	if got := d.Orientation(); got != OrientationUnknown {
		t.Errorf("Orientation() = %q, want %q", got, OrientationUnknown)
	}

	// HandleOrientation still has to pick a class, and portrait is its
	// fallback branch, so the class and the cached value disagree here.
	if got := d.HandleOrientation(""); got != " portrait" {
		t.Errorf("HandleOrientation(\"\") = %q, want %q", got, " portrait")
	}
	if got := d.Orientation(); got != OrientationUnknown {
		t.Errorf("Orientation() = %q after HandleOrientation, want %q", got, OrientationUnknown)
	}
}

func TestHandleOrientationSwapsClasses(t *testing.T) {
	t.Run("portrait replaces landscape", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 400, InnerHeight: 800})
		if got := d.HandleOrientation(" mobile landscape"); got != "mobile portrait" {
			t.Errorf("HandleOrientation() = %q, want %q", got, "mobile portrait")
		}
	})

	t.Run("landscape replaces portrait", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 800, InnerHeight: 400})
		if got := d.HandleOrientation(" mobile portrait"); got != "mobile landscape" {
			t.Errorf("HandleOrientation() = %q, want %q", got, "mobile landscape")
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 400, InnerHeight: 800})
		once := d.HandleOrientation(" mobile")
		twice := d.HandleOrientation(once)
		if once != twice {
			t.Errorf("HandleOrientation is not idempotent: %q then %q", once, twice)
		}
	})
}

func TestHandleOrientationRefreshesCache(t *testing.T) {
	d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 400, InnerHeight: 800})
	if got := d.Orientation(); got != OrientationPortrait {
		t.Fatalf("Orientation() = %q, want %q", got, OrientationPortrait)
	}

	env := d.Env()
	env.InnerWidth, env.InnerHeight = 800, 400
	d.SetEnv(env)

	if got := d.Orientation(); got != OrientationPortrait {
		t.Errorf("Orientation() = %q before HandleOrientation, want the stale %q", got, OrientationPortrait)
	}

	d.HandleOrientation("")

	if got := d.Orientation(); got != OrientationLandscape {
		t.Errorf("Orientation() = %q after HandleOrientation, want %q", got, OrientationLandscape)
	}
}

func TestOnChangeOrientationCallbacks(t *testing.T) {
	t.Run("callbacks fire in registration order", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 800, InnerHeight: 400})

		var order []string
		d.OnChangeOrientation(func(DeviceOrientation) { order = append(order, "first") })
		d.OnChangeOrientation(func(DeviceOrientation) { order = append(order, "second") })
		d.OnChangeOrientation(func(DeviceOrientation) { order = append(order, "third") })

		d.HandleOrientation("")

		if len(order) != 3 || order[0] != "first" || order[1] != "second" || order[2] != "third" {
			t.Errorf("callback order = %v, want [first second third]", order)
		}
	})

	t.Run("callbacks receive the new orientation", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 800, InnerHeight: 400})

		var got DeviceOrientation
		d.OnChangeOrientation(func(o DeviceOrientation) { got = o })
		d.HandleOrientation("")
		if got != OrientationLandscape {
			t.Errorf("callback received %q, want %q", got, OrientationLandscape)
		}

		env := d.Env()
		env.InnerWidth, env.InnerHeight = 400, 800
		d.SetEnv(env)
		d.HandleOrientation("")
		if got != OrientationPortrait {
			t.Errorf("callback received %q, want %q", got, OrientationPortrait)
		}
	})

	t.Run("a nil callback is ignored", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 400, InnerHeight: 800})
		d.OnChangeOrientation(nil)

		called := false
		d.OnChangeOrientation(func(DeviceOrientation) { called = true })

		d.HandleOrientation("")

		if !called {
			t.Error("the surviving callback did not fire")
		}
	})

	t.Run("registering does not fire immediately", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 400, InnerHeight: 800})
		called := false
		d.OnChangeOrientation(func(DeviceOrientation) { called = true })
		if called {
			t.Error("callback fired on registration; it must wait for HandleOrientation")
		}
	})

	t.Run("a callback may call back into the device", func(t *testing.T) {
		d := New(Env{UserAgent: "Mozilla/5.0", InnerWidth: 400, InnerHeight: 800})

		var observed DeviceOrientation
		d.OnChangeOrientation(func(DeviceOrientation) {
			observed = d.Orientation()
			_ = d.Type()
			_ = d.Mobile()
		})

		d.HandleOrientation("")

		if observed == "" {
			t.Error("re-entrant call returned an empty orientation")
		}
	})
}

func TestOrientationEventName(t *testing.T) {
	withHook := New(Env{UserAgent: "Mozilla/5.0", HasOnOrientationChange: true})
	if got := withHook.OrientationEventName(); got != EventOrientationChange {
		t.Errorf("OrientationEventName() = %q, want %q", got, EventOrientationChange)
	}

	withoutHook := New(Env{UserAgent: "Mozilla/5.0"})
	if got := withoutHook.OrientationEventName(); got != EventResize {
		t.Errorf("OrientationEventName() = %q, want %q", got, EventResize)
	}

	if EventOrientationChange == EventResize {
		t.Fatal("the two branches must select different window events")
	}
}

// TestConcurrentAccess exercises the mutex under the race detector.
func TestConcurrentAccess(t *testing.T) {
	d := New(Env{UserAgent: iPhoneUA, InnerWidth: 400, InnerHeight: 800})
	d.OnChangeOrientation(func(DeviceOrientation) {})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = d.Type()
				_ = d.OS()
				_ = d.Orientation()
				_ = d.Mobile()
				_ = d.ClassNames("")
				d.HandleOrientation("")
				d.OnChangeOrientation(func(DeviceOrientation) {})
			}
		}()
	}
	wg.Wait()
}
