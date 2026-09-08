//go:build js && wasm

package main

import (
	"syscall/js"
	"testing"

	device "github.com/matthewhudson/current-device-go"
)

const iPhoneUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"

// fakeWindow is a minimal stand-in for the browser globals the binding reads.
// Node has no DOM and its `navigator` is not writable, so the window object has
// to be synthesised rather than patched onto globalThis.
type fakeWindow struct {
	value     js.Value
	listeners map[string][]js.Value
}

func newFakeWindow(userAgent string) *fakeWindow {
	object := js.Global().Get("Object")

	w := &fakeWindow{value: object.New(), listeners: map[string][]js.Value{}}

	navigator := object.New()
	navigator.Set("userAgent", userAgent)
	navigator.Set("platform", "iPhone")
	navigator.Set("maxTouchPoints", 5)
	w.value.Set("navigator", navigator)

	documentElement := object.New()
	documentElement.Set("className", "")
	document := object.New()
	document.Set("documentElement", documentElement)
	w.value.Set("document", document)

	location := object.New()
	location.Set("protocol", "https:")
	w.value.Set("location", location)

	w.value.Set("innerWidth", 390)
	w.value.Set("innerHeight", 844)

	w.value.Set("addEventListener", js.FuncOf(func(_ js.Value, args []js.Value) any {
		w.listeners[args[0].String()] = append(w.listeners[args[0].String()], args[1])
		return js.Undefined()
	}))

	return w
}

func (w *fakeWindow) setScreenOrientation(orientationType string) {
	orientation := js.Global().Get("Object").New()
	orientation.Set("type", orientationType)
	screen := js.Global().Get("Object").New()
	screen.Set("orientation", orientation)
	w.value.Set("screen", screen)
	w.value.Set("onorientationchange", js.Null())
}

func (w *fakeWindow) className() string {
	return w.value.Get("document").Get("documentElement").Get("className").String()
}

func (w *fakeWindow) resize(width, height int) {
	w.value.Set("innerWidth", width)
	w.value.Set("innerHeight", height)
}

func (w *fakeWindow) dispatch(t *testing.T, event string) {
	t.Helper()

	registered := w.listeners[event]
	if len(registered) == 0 {
		t.Fatalf("no listener registered for %q", event)
	}

	for _, listener := range registered {
		listener.Invoke(js.Undefined())
	}
}

func install(userAgent string) (*fakeWindow, js.Value) {
	w := newFakeWindow(userAgent)
	newBrowserDevice(w.value).install()

	return w, w.value.Get("device")
}

func TestInstallPublishesGlobalDevice(t *testing.T) {
	_, exported := install(iPhoneUA)

	if exported.Type() != js.TypeObject {
		t.Fatalf("window.device type = %v, want object", exported.Type())
	}

	for _, name := range append(device.PredicateNames(), "onChangeOrientation", "noConflict") {
		if got := exported.Get(name).Type(); got != js.TypeFunction {
			t.Errorf("device.%s type = %v, want function", name, got)
		}
	}

	for _, name := range []string{"type", "os", "orientation"} {
		if got := exported.Get(name).Type(); got != js.TypeString {
			t.Errorf("device.%s type = %v, want string", name, got)
		}
	}
}

func TestExportedPredicatesMatchCore(t *testing.T) {
	_, exported := install(iPhoneUA)

	want := map[string]bool{
		"ios": true, "iphone": true, "mobile": true, "portrait": true,
		"ipad": false, "android": false, "desktop": false, "landscape": false,
	}

	for name, expected := range want {
		if got := exported.Call(name).Bool(); got != expected {
			t.Errorf("device.%s() = %v, want %v", name, got, expected)
		}
	}

	if got := exported.Get("os").String(); got != string(device.OSIOS) {
		t.Errorf("device.os = %q, want %q", got, device.OSIOS)
	}

	if got := exported.Get("type").String(); got != string(device.TypeMobile) {
		t.Errorf("device.type = %q, want %q", got, device.TypeMobile)
	}
}

func TestInstallStampsDetectionClasses(t *testing.T) {
	w, _ := install(iPhoneUA)

	if got, want := w.className(), "ios iphone mobile portrait"; got != want {
		t.Errorf("className = %q, want %q", got, want)
	}
}

func TestNoConflictRestoresPreviousGlobal(t *testing.T) {
	w := newFakeWindow(iPhoneUA)
	sentinel := "previous-owner"
	w.value.Set("device", sentinel)

	newBrowserDevice(w.value).install()
	exported := w.value.Get("device")

	if returned := exported.Call("noConflict"); !returned.Equal(exported) {
		t.Error("noConflict() did not return the device object")
	}

	if got := w.value.Get("device").String(); got != sentinel {
		t.Errorf("window.device = %q, want %q", got, sentinel)
	}
}

func TestNoConflictRestoresUndefined(t *testing.T) {
	w, exported := install(iPhoneUA)

	exported.Call("noConflict")

	if got := w.value.Get("device"); got.Type() != js.TypeUndefined {
		t.Errorf("window.device type = %v, want undefined", got.Type())
	}
}

func TestOrientationChangeUpdatesClassesAndCallbacks(t *testing.T) {
	w, exported := install(iPhoneUA)

	var received []string
	exported.Call("onChangeOrientation", js.FuncOf(func(_ js.Value, args []js.Value) any {
		received = append(received, args[0].String())
		return js.Undefined()
	}))

	if len(received) != 0 {
		t.Fatalf("callback fired on registration: %v", received)
	}

	w.resize(844, 390)
	w.dispatch(t, "resize")

	if len(received) != 1 || received[0] != string(device.OrientationLandscape) {
		t.Fatalf("callback received %v, want [landscape]", received)
	}

	if got, want := w.className(), "ios iphone mobile landscape"; got != want {
		t.Errorf("className = %q, want %q", got, want)
	}

	if got := exported.Get("orientation").String(); got != string(device.OrientationLandscape) {
		t.Errorf("device.orientation = %q, want landscape", got)
	}

	if !exported.Call("landscape").Bool() || exported.Call("portrait").Bool() {
		t.Error("live predicates disagree with the dispatched orientation")
	}
}

func TestNonFunctionOrientationCallbackIsIgnored(t *testing.T) {
	w, exported := install(iPhoneUA)

	exported.Call("onChangeOrientation", "not a function")
	exported.Call("onChangeOrientation")

	w.resize(844, 390)
	w.dispatch(t, "resize")
}

func TestOrientationChangeEventPreferredWhenSupported(t *testing.T) {
	w := newFakeWindow(iPhoneUA)
	w.setScreenOrientation("landscape-primary")
	newBrowserDevice(w.value).install()

	if len(w.listeners["orientationchange"]) != 1 {
		t.Fatalf("listeners = %v, want one orientationchange listener", w.listeners)
	}

	if got := w.value.Get("device").Get("orientation").String(); got != string(device.OrientationLandscape) {
		t.Errorf("device.orientation = %q, want landscape", got)
	}
}

func TestReadEnvToleratesMissingGlobals(t *testing.T) {
	bare := js.Global().Get("Object").New()
	bare.Set("addEventListener", js.FuncOf(func(js.Value, []js.Value) any { return js.Undefined() }))

	newBrowserDevice(bare).install()

	exported := bare.Get("device")
	if got := exported.Get("os").String(); got != string(device.OSUnknown) {
		t.Errorf("device.os = %q, want unknown", got)
	}

	if got := exported.Get("orientation").String(); got != string(device.OrientationUnknown) {
		t.Errorf("device.orientation = %q, want unknown", got)
	}

	if !exported.Call("desktop").Bool() {
		t.Error("an empty user agent should resolve to desktop")
	}
}

func TestCordovaRequiresFileProtocol(t *testing.T) {
	w := newFakeWindow(iPhoneUA)
	w.value.Set("cordova", js.Global().Get("Object").New())
	newBrowserDevice(w.value).install()

	if w.value.Get("device").Call("cordova").Bool() {
		t.Error("cordova() should be false over https:")
	}

	w.value.Get("location").Set("protocol", "file:")

	if !w.value.Get("device").Call("cordova").Bool() {
		t.Error("cordova() should be true over file:")
	}
}

// goPolyfillProcess reproduces the exact stub that Go's wasm_exec.js assigns to
// globalThis.process in a browser: shell helpers only, no versions and no env.
func goPolyfillProcess() js.Value {
	process := js.Global().Get("Object").New()
	for _, name := range []string{
		"getuid", "getgid", "geteuid", "getegid", "getgroups",
		"pid", "ppid", "umask", "cwd", "chdir",
	} {
		process.Set(name, js.FuncOf(func(js.Value, []js.Value) any { return nil }))
	}

	return process
}

func nodeWebkitProcess() js.Value {
	versions := js.Global().Get("Object").New()
	versions.Set("node", "20.11.0")
	versions.Set("nw", "0.85.0")

	process := goPolyfillProcess()
	process.Set("versions", versions)
	process.Set("env", js.Global().Get("Object").New())

	return process
}

func TestNodeWebkitDetection(t *testing.T) {
	tests := []struct {
		name    string
		process func() js.Value
		want    bool
	}{
		{name: "absent in a plain browser", process: nil, want: false},
		{name: "go wasm_exec.js polyfill is ignored", process: goPolyfillProcess, want: false},
		{name: "real nw.js process", process: nodeWebkitProcess, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := newFakeWindow(iPhoneUA)
			if test.process != nil {
				w.value.Set("process", test.process())
			}
			newBrowserDevice(w.value).install()

			if got := w.value.Get("device").Call("nodeWebkit").Bool(); got != test.want {
				t.Errorf("nodeWebkit() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestGoProcessPolyfillDoesNotHijackTheClassCascade guards the case the macOS
// browser check hid: `macos` and `windows` are tested before `nodeWebkit`, so
// only a Linux desktop or a television exposes the polyfill's effect on classes.
func TestGoProcessPolyfillDoesNotHijackTheClassCascade(t *testing.T) {
	const linuxUA = "mozilla/5.0 (x11; linux x86_64) applewebkit/537.36 (khtml, like gecko) chrome/133.0.0.0 safari/537.36"

	w := newFakeWindow(linuxUA)
	w.value.Set("process", goPolyfillProcess())
	newBrowserDevice(w.value).install()

	if got, want := w.className(), "desktop portrait"; got != want {
		t.Errorf("className = %q, want %q", got, want)
	}
}

func TestMissingClassNameIsTreatedAsEmpty(t *testing.T) {
	w := newFakeWindow(iPhoneUA)
	w.value.Get("document").Get("documentElement").Delete("className")
	newBrowserDevice(w.value).install()

	if got, want := w.className(), "ios iphone mobile portrait"; got != want {
		t.Errorf("className = %q, want %q", got, want)
	}
}

func TestLegacyOrientationAngle(t *testing.T) {
	w := newFakeWindow(iPhoneUA)
	w.value.Set("orientation", -90)
	newBrowserDevice(w.value).install()

	exported := w.value.Get("device")
	if !exported.Call("landscape").Bool() {
		t.Error("window.orientation of -90 should report landscape")
	}
}
