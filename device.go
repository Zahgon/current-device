package device

import (
	"strings"
	"sync"
)

// DeviceType is the coarse form factor of the current device.
type DeviceType string

// Recognised values for DeviceType.
const (
	TypeMobile  DeviceType = "mobile"
	TypeTablet  DeviceType = "tablet"
	TypeDesktop DeviceType = "desktop"
	TypeUnknown DeviceType = "unknown"
)

// DeviceOS is the detected operating system of the current device.
type DeviceOS string

// Recognised values for DeviceOS.
const (
	OSIOS        DeviceOS = "ios"
	OSIPhone     DeviceOS = "iphone"
	OSIPad       DeviceOS = "ipad"
	OSIPod       DeviceOS = "ipod"
	OSAndroid    DeviceOS = "android"
	OSBlackberry DeviceOS = "blackberry"
	OSMacOS      DeviceOS = "macos"
	OSWindows    DeviceOS = "windows"
	OSFxOS       DeviceOS = "fxos"
	OSMeeGo      DeviceOS = "meego"
	OSTelevision DeviceOS = "television"
	OSHarmonyOS  DeviceOS = "harmonyos"
	OSUnknown    DeviceOS = "unknown"
)

// DeviceOrientation is the current screen orientation.
type DeviceOrientation string

// Recognised values for DeviceOrientation.
const (
	OrientationLandscape DeviceOrientation = "landscape"
	OrientationPortrait  DeviceOrientation = "portrait"
	OrientationUnknown   DeviceOrientation = "unknown"
)

// OrientationChangeCallback is invoked with the new orientation whenever the
// device orientation changes. It corresponds to the TypeScript type
// `(newOrientation: 'landscape' | 'portrait') => void`, so it is only ever
// called with OrientationLandscape or OrientationPortrait, never
// OrientationUnknown.
type OrientationChangeCallback func(newOrientation DeviceOrientation)

// televisionDevices is the list of detectable television devices. The order is
// significant and identical to the TypeScript source: Television returns on the
// first match.
var televisionDevices = []string{
	"googletv",
	"viera",
	"smarttv",
	"internet.tv",
	"netcast",
	"nettv",
	"appletv",
	"boxee",
	"kylo",
	"roku",
	"dlnadoc",
	"pov_tv",
	"hbbtv",
	"ce-html",
}

// Device detects the operating system, form factor and orientation of a client
// from an Env snapshot.
//
// It is the Go equivalent of the `device` singleton exported by the TypeScript
// package, with one deliberate architectural difference: it is an ordinary
// value constructed from an explicit Env rather than a module-scope singleton
// bound to browser globals. Detection methods are pure functions of the Env.
//
// A Device is safe for concurrent use.
type Device struct {
	mu  sync.RWMutex
	env Env

	deviceType  DeviceType
	os          DeviceOS
	orientation DeviceOrientation

	changeOrientationList []OrientationChangeCallback
}

// New returns a Device for the supplied environment snapshot.
//
// As in the TypeScript module, the Type and OS properties are resolved once at
// construction time and then cached; Orientation is resolved at construction
// and re-resolved by every call to HandleOrientation.
func New(env Env) *Device {
	d := &Device{env: env.normalize()}
	d.deviceType = d.resolveType()
	d.os = d.resolveOS()
	d.orientation = d.resolveOrientation()
	return d
}

// NewFromUserAgent returns a Device for a user agent string with every other
// environment field left at its zero value.
//
// This is the constructor to use on a server, e.g.
// device.NewFromUserAgent(r.UserAgent()). Note that orientation cannot be
// determined from a user agent alone, so Orientation reports
// OrientationUnknown: with a zero viewport the height/width ratio is NaN and,
// exactly as in JavaScript, every NaN comparison is false.
func NewFromUserAgent(ua string) *Device {
	return New(Env{UserAgent: ua})
}

// Env returns the current environment snapshot.
func (d *Device) Env() Env {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.env
}

// SetEnv replaces the environment snapshot, lower-casing the user agent.
//
// It is used by the WebAssembly layer to refresh viewport and orientation
// readings before recomputing orientation, because the TypeScript original
// re-read `window.innerWidth`, `window.innerHeight` and `screen.orientation`
// live on every orientation event.
//
// The cached Type and OS values are deliberately NOT recomputed: the original
// resolved them exactly once, at module import, and never again.
func (d *Device) SetEnv(env Env) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.env = env.normalize()
}

// find reports whether the lower-cased user agent contains needle. It is the
// Go equivalent of the private `find` helper, which itself wraps
// `userAgent.indexOf(needle) !== -1`.
func (d *Device) find(needle string) bool {
	d.mu.RLock()
	ua := d.env.UserAgent
	d.mu.RUnlock()
	return strings.Contains(ua, needle)
}

// Operating system detection
// --------------------------

// MacOS reports whether the device runs macOS.
func (d *Device) MacOS() bool { return d.find("mac") && !d.IOS() }

// IOS reports whether the device runs iOS or iPadOS.
func (d *Device) IOS() bool { return d.IPhone() || d.IPod() || d.IPad() }

// IPhone reports whether the device is an iPhone.
func (d *Device) IPhone() bool { return !d.Windows() && d.find("iphone") }

// IPod reports whether the device is an iPod touch.
func (d *Device) IPod() bool { return d.find("ipod") }

// IPad reports whether the device is an iPad.
//
// iPadOS 13 and later request the desktop site by default, so their user agent
// is indistinguishable from a Mac. The original disambiguates using
// navigator.platform and navigator.maxTouchPoints, and so does this port.
func (d *Device) IPad() bool {
	d.mu.RLock()
	iPadOS13Up := d.env.Platform == "MacIntel" && d.env.MaxTouchPoints > 1
	d.mu.RUnlock()
	return d.find("ipad") || iPadOS13Up
}

// Android reports whether the device runs Android.
func (d *Device) Android() bool { return !d.Windows() && d.find("android") }

// AndroidPhone reports whether the device is an Android phone.
func (d *Device) AndroidPhone() bool { return d.Android() && d.find("mobile") }

// AndroidTablet reports whether the device is an Android tablet.
func (d *Device) AndroidTablet() bool { return d.Android() && !d.find("mobile") }

// Blackberry reports whether the device runs BlackBerry OS or BlackBerry 10.
func (d *Device) Blackberry() bool { return d.find("blackberry") || d.find("bb10") }

// BlackberryPhone reports whether the device is a BlackBerry phone.
func (d *Device) BlackberryPhone() bool { return d.Blackberry() && !d.find("tablet") }

// BlackberryTablet reports whether the device is a BlackBerry tablet.
func (d *Device) BlackberryTablet() bool { return d.Blackberry() && d.find("tablet") }

// Windows reports whether the device runs any version of Windows.
func (d *Device) Windows() bool { return d.find("windows") }

// WindowsPhone reports whether the device is a Windows phone.
func (d *Device) WindowsPhone() bool { return d.Windows() && d.find("phone") }

// WindowsTablet reports whether the device is a Windows tablet.
func (d *Device) WindowsTablet() bool {
	return d.Windows() && (d.find("touch") && !d.WindowsPhone())
}

// FxOS reports whether the device runs Firefox OS.
//
// The needles are "(mobile" and "(tablet" with a leading parenthesis, and
// " rv:" with a leading space. Those exact spellings are load-bearing: they are
// what stop desktop Firefox, whose user agent also contains "rv:", from being
// misdetected as Firefox OS.
func (d *Device) FxOS() bool {
	return (d.find("(mobile") || d.find("(tablet")) && d.find(" rv:")
}

// FxOSPhone reports whether the device is a Firefox OS phone.
func (d *Device) FxOSPhone() bool { return d.FxOS() && d.find("mobile") }

// FxOSTablet reports whether the device is a Firefox OS tablet.
func (d *Device) FxOSTablet() bool { return d.FxOS() && d.find("tablet") }

// MeeGo reports whether the device runs MeeGo.
func (d *Device) MeeGo() bool { return d.find("meego") }

// HarmonyOS reports whether the device runs HarmonyOS.
func (d *Device) HarmonyOS() bool { return d.find("harmonyos") }

// Television reports whether the device is a connected television.
func (d *Device) Television() bool {
	i := 0
	for i < len(televisionDevices) {
		if d.find(televisionDevices[i]) {
			return true
		}
		i++
	}
	return false
}

// Runtime detection
// -----------------

// Cordova reports whether the page is running inside an Apache Cordova
// container, i.e. a `window.cordova` object served over the file: protocol.
func (d *Device) Cordova() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.env.HasCordova && d.env.LocationProtocol == "file:"
}

// NodeWebkit reports whether the page is running inside NW.js (node-webkit),
// probed via `typeof window.process === 'object'`.
//
// Beware: any environment that exposes a Node.js `process` global satisfies
// this probe. Under jsdom the original returns true, which is why the upstream
// test suite special-cases it. That behaviour is preserved here.
func (d *Device) NodeWebkit() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.env.HasProcessObject
}

// Form factor detection
// ---------------------

// Mobile reports whether the device is a phone.
func (d *Device) Mobile() bool {
	return d.AndroidPhone() ||
		d.IPhone() ||
		d.IPod() ||
		d.WindowsPhone() ||
		d.BlackberryPhone() ||
		d.FxOSPhone() ||
		d.MeeGo()
}

// Tablet reports whether the device is a tablet.
func (d *Device) Tablet() bool {
	return d.IPad() ||
		d.AndroidTablet() ||
		d.BlackberryTablet() ||
		d.WindowsTablet() ||
		d.FxOSTablet()
}

// Desktop reports whether the device is neither a tablet nor a phone.
//
// Note that this makes Desktop the fallback for anything unrecognised,
// televisions included: a smart TV reports OS "television" and Type "desktop".
func (d *Device) Desktop() bool { return !d.Tablet() && !d.Mobile() }

// Orientation detection
// ---------------------

// Portrait reports whether the device is held in portrait orientation.
//
// Three strategies are tried in the same order as the original: the Screen
// Orientation API, the legacy iOS window.orientation angle, and finally the
// viewport aspect ratio.
func (d *Device) Portrait() bool {
	d.mu.RLock()
	env := d.env
	d.mu.RUnlock()

	if env.HasScreenOrientation && env.HasOnOrientationChange {
		return strings.Contains(env.ScreenOrientationType, "portrait")
	}
	if d.IOS() && env.HasWindowOrientation {
		return abs(env.WindowOrientation) != 90
	}
	return env.InnerHeight/env.InnerWidth > 1
}

// Landscape reports whether the device is held in landscape orientation.
//
// Portrait and Landscape are independent probes rather than complements: on a
// perfectly square viewport the ratio is exactly 1, so both return false and
// Orientation reports OrientationUnknown. That quirk is inherited from the
// TypeScript original and preserved deliberately.
func (d *Device) Landscape() bool {
	d.mu.RLock()
	env := d.env
	d.mu.RUnlock()

	if env.HasScreenOrientation && env.HasOnOrientationChange {
		return strings.Contains(env.ScreenOrientationType, "landscape")
	}
	if d.IOS() && env.HasWindowOrientation {
		return abs(env.WindowOrientation) == 90
	}
	return env.InnerHeight/env.InnerWidth < 1
}

// NoConflict returns the Device itself.
//
// In the browser this is the hook that restores the previous value of the
// global `device` variable; see cmd/wasm, which wires the global assignment.
// In pure Go there is no global to restore, so this only returns the receiver,
// matching the `return this` of the original.
func (d *Device) NoConflict() *Device { return d }

// abs returns the absolute value of n, mirroring Math.abs on window.orientation.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
