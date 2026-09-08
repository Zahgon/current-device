package device

// predicates is the canonical registry of every boolean detection method,
// keyed by the exact camelCase name used by the TypeScript API.
//
// The original resolved `device.type` and `device.os` by indexing the device
// object with strings (`device[arr[i]]`), and its test suite drove assertions
// the same way. Keeping a single registry preserves that dynamic dispatch,
// guarantees the Go methods and the JavaScript bindings can never drift apart,
// and gives the WebAssembly layer one table to export from.
var predicates = map[string]func(*Device) bool{
	"macos":            (*Device).MacOS,
	"ios":              (*Device).IOS,
	"iphone":           (*Device).IPhone,
	"ipod":             (*Device).IPod,
	"ipad":             (*Device).IPad,
	"android":          (*Device).Android,
	"androidPhone":     (*Device).AndroidPhone,
	"androidTablet":    (*Device).AndroidTablet,
	"blackberry":       (*Device).Blackberry,
	"blackberryPhone":  (*Device).BlackberryPhone,
	"blackberryTablet": (*Device).BlackberryTablet,
	"windows":          (*Device).Windows,
	"windowsPhone":     (*Device).WindowsPhone,
	"windowsTablet":    (*Device).WindowsTablet,
	"fxos":             (*Device).FxOS,
	"fxosPhone":        (*Device).FxOSPhone,
	"fxosTablet":       (*Device).FxOSTablet,
	"meego":            (*Device).MeeGo,
	"harmonyos":        (*Device).HarmonyOS,
	"television":       (*Device).Television,
	"cordova":          (*Device).Cordova,
	"nodeWebkit":       (*Device).NodeWebkit,
	"mobile":           (*Device).Mobile,
	"tablet":           (*Device).Tablet,
	"desktop":          (*Device).Desktop,
	"portrait":         (*Device).Portrait,
	"landscape":        (*Device).Landscape,
}

// PredicateNames lists the camelCase names of every detection method, in the
// declaration order of the TypeScript `Device` interface.
//
// The order is fixed so that callers iterating the API — the WebAssembly
// bindings and the conformance tests — do so deterministically.
func PredicateNames() []string {
	return []string{
		"macos", "ios", "iphone", "ipod", "ipad",
		"android", "androidPhone", "androidTablet",
		"blackberry", "blackberryPhone", "blackberryTablet",
		"windows", "windowsPhone", "windowsTablet",
		"fxos", "fxosPhone", "fxosTablet",
		"meego", "harmonyos", "television",
		"cordova", "nodeWebkit",
		"mobile", "tablet", "desktop",
		"portrait", "landscape",
	}
}

// Predicate returns the detection method registered under its TypeScript
// camelCase name, reproducing the `device[name]()` dispatch of the original.
// The second return value reports whether the name is known.
func (d *Device) Predicate(name string) (func() bool, bool) {
	fn, ok := predicates[name]
	if !ok {
		return nil, false
	}
	return func() bool { return fn(d) }, true
}

// typeMatchOrder, osMatchOrder and orientationMatchOrder are the argument
// arrays passed to `findMatch` by the TypeScript source. Their order decides
// which label wins when several predicates are true, so it must not change.
var (
	typeMatchOrder = []string{"mobile", "tablet", "desktop"}

	// Note that "ios" precedes "iphone", "ipad" and "ipod". Every iOS device
	// therefore reports OS "ios" and never the more specific label; use
	// IPhone, IPad or IPod for that. Note too that "harmonyos" precedes
	// "android", because HarmonyOS user agents also contain "android".
	osMatchOrder = []string{
		"ios",
		"iphone",
		"ipad",
		"ipod",
		"harmonyos",
		"android",
		"blackberry",
		"macos",
		"windows",
		"fxos",
		"meego",
		"television",
	}

	orientationMatchOrder = []string{"portrait", "landscape"}
)

// findMatch returns the first name in order whose predicate is true, or
// "unknown" if none is. It is the Go equivalent of the private `findMatch`
// helper.
func (d *Device) findMatch(order []string) string {
	for _, name := range order {
		if fn, ok := predicates[name]; ok && fn(d) {
			return name
		}
	}
	return "unknown"
}

func (d *Device) resolveType() DeviceType {
	return DeviceType(d.findMatch(typeMatchOrder))
}

func (d *Device) resolveOS() DeviceOS {
	return DeviceOS(d.findMatch(osMatchOrder))
}

func (d *Device) resolveOrientation() DeviceOrientation {
	return DeviceOrientation(d.findMatch(orientationMatchOrder))
}

// Type returns the cached device type, equivalent to the `device.type`
// property. It is resolved once, when the Device is constructed.
func (d *Device) Type() DeviceType {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.deviceType
}

// OS returns the cached operating system, equivalent to the `device.os`
// property. It is resolved once, when the Device is constructed.
func (d *Device) OS() DeviceOS {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.os
}

// Orientation returns the cached orientation, equivalent to the
// `device.orientation` property. It is resolved when the Device is constructed
// and refreshed by every call to HandleOrientation.
func (d *Device) Orientation() DeviceOrientation {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.orientation
}

// setOrientationCache re-resolves the cached orientation, mirroring the
// private `setOrientationCache` helper.
func (d *Device) setOrientationCache() {
	orientation := d.resolveOrientation()
	d.mu.Lock()
	d.orientation = orientation
	d.mu.Unlock()
}
