//go:build js && wasm

// Command wasm is the browser entry point for current-device-go. It reproduces
// every import-time side effect of the original TypeScript module: it publishes
// a `device` global, stamps detection classes onto the <html> element, and keeps
// the orientation classes in sync with the viewport.
package main

import (
	"syscall/js"

	device "github.com/matthewhudson/current-device-go"
)

func main() {
	newBrowserDevice(js.Global()).install()

	// syscall/js callbacks are only reachable while main is alive.
	select {}
}

// browserDevice binds a device.Device to the live browser globals.
type browserDevice struct {
	dev      *device.Device
	window   js.Value
	docEl    js.Value
	exported js.Value
	previous js.Value
}

// newBrowserDevice binds to the supplied window object. Production passes
// js.Global(); tests pass a stand-in so the DOM contract can be exercised.
func newBrowserDevice(window js.Value) *browserDevice {
	b := &browserDevice{
		window: window,
		// Captured before the global is overwritten so noConflict can restore it.
		previous: window.Get("device"),
		exported: js.Global().Get("Object").New(),
	}

	if document := window.Get("document"); document.Truthy() {
		b.docEl = document.Get("documentElement")
	}

	b.dev = device.New(b.readEnv())

	return b
}

// install performs the side effects in the same order as the TypeScript module:
// publish the global, apply the detection classes, subscribe to orientation
// changes, then resolve the cached type/os/orientation values.
func (b *browserDevice) install() {
	b.exportPredicates()
	b.exportControls()
	b.window.Set("device", b.exported)

	b.applyClasses()

	listener := js.FuncOf(func(js.Value, []js.Value) any {
		b.handleOrientation()
		return js.Undefined()
	})
	b.window.Call("addEventListener", b.dev.OrientationEventName(), listener, false)

	b.handleOrientation()

	b.exported.Set("type", string(b.dev.Type()))
	b.exported.Set("os", string(b.dev.OS()))
	b.exported.Set("orientation", string(b.dev.Orientation()))
}

// exportPredicates mirrors the 27 boolean methods under their original
// TypeScript names. Each call re-reads the browser globals first because
// portrait/landscape depend on the live viewport.
func (b *browserDevice) exportPredicates() {
	for _, name := range device.PredicateNames() {
		predicate, ok := b.dev.Predicate(name)
		if !ok {
			continue
		}

		b.exported.Set(name, js.FuncOf(func(js.Value, []js.Value) any {
			b.dev.SetEnv(b.readEnv())
			return predicate()
		}))
	}
}

func (b *browserDevice) exportControls() {
	b.exported.Set("onChangeOrientation", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 || args[0].Type() != js.TypeFunction {
			return js.Undefined()
		}

		callback := args[0]
		b.dev.OnChangeOrientation(func(orientation device.DeviceOrientation) {
			callback.Invoke(string(orientation))
		})

		return js.Undefined()
	}))

	b.exported.Set("noConflict", js.FuncOf(func(js.Value, []js.Value) any {
		b.window.Set("device", b.previous)
		return b.exported
	}))
}

func (b *browserDevice) applyClasses() {
	if !b.docEl.Truthy() {
		return
	}

	b.docEl.Set("className", b.dev.ClassNames(b.className()))
}

func (b *browserDevice) handleOrientation() {
	b.dev.SetEnv(b.readEnv())
	updated := b.dev.HandleOrientation(b.className())

	if b.docEl.Truthy() {
		b.docEl.Set("className", updated)
	}

	b.exported.Set("orientation", string(b.dev.Orientation()))
}

func (b *browserDevice) className() string {
	if !b.docEl.Truthy() {
		return ""
	}

	return stringOr(b.docEl.Get("className"), "")
}

// readEnv snapshots the browser globals the detection logic depends on. Every
// lookup is guarded because Cordova, Node-Webkit, the Screen Orientation API and
// the legacy window.orientation angle are all absent on most platforms.
func (b *browserDevice) readEnv() device.Env {
	env := device.Env{
		HasOnOrientationChange: hasOwnProperty(b.window, "onorientationchange"),
		HasWindowOrientation:   hasOwnProperty(b.window, "orientation"),
		WindowOrientation:      int(numberOr(b.window.Get("orientation"), 0)),
		InnerWidth:             numberOr(b.window.Get("innerWidth"), 0),
		InnerHeight:            numberOr(b.window.Get("innerHeight"), 0),
		HasCordova:             b.window.Get("cordova").Truthy(),
		HasProcessObject:       hasRuntimeProcessObject(b.window),
	}

	if navigator := b.window.Get("navigator"); navigator.Truthy() {
		env.UserAgent = stringOr(navigator.Get("userAgent"), "")
		env.Platform = stringOr(navigator.Get("platform"), "")
		env.MaxTouchPoints = int(numberOr(navigator.Get("maxTouchPoints"), 0))
	}

	if screen := b.window.Get("screen"); screen.Truthy() {
		if orientation := screen.Get("orientation"); orientation.Truthy() {
			env.HasScreenOrientation = true
			env.ScreenOrientationType = stringOr(orientation.Get("type"), "")
		}
	}

	if location := b.window.Get("location"); location.Truthy() {
		env.LocationProtocol = stringOr(location.Get("protocol"), "")
	}

	return env
}

// hasRuntimeProcessObject reports whether a genuine Node/NW.js `process` global
// is present, mirroring the source's `typeof window.process === 'object'`.
//
// The extra `versions` requirement exists because Go's own wasm_exec.js installs
// a stub `process` polyfill (getuid/cwd/chdir/... only) into every browser before
// this binary starts. Without the guard, that runtime artifact would make
// nodeWebkit() true in a plain browser, and on Linux or a television it would
// stamp the `node-webkit` class instead of `desktop`/`television`. Every real
// Node and NW.js process object carries `versions`; the polyfill does not.
func hasRuntimeProcessObject(window js.Value) bool {
	process := window.Get("process")
	if process.Type() != js.TypeObject {
		return false
	}

	return process.Get("versions").Type() == js.TypeObject
}

// hasOwnProperty reproduces `Object.prototype.hasOwnProperty.call(window, prop)`.
// A truthiness check is not equivalent: window.orientation is 0 in portrait.
func hasOwnProperty(value js.Value, property string) bool {
	return js.Global().
		Get("Object").Get("prototype").Get("hasOwnProperty").
		Call("call", value, property).Bool()
}

func stringOr(value js.Value, fallback string) string {
	if value.Type() == js.TypeString {
		return value.String()
	}

	return fallback
}

func numberOr(value js.Value, fallback float64) float64 {
	if value.Type() == js.TypeNumber {
		return value.Float()
	}

	return fallback
}
