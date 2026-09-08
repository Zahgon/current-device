package device

import "strings"

// Env is an explicit, injectable snapshot of every browser global that the
// original TypeScript implementation read directly from `window`, `navigator`,
// `screen`, `document` and `location`.
//
// The TypeScript module read those globals at module scope, which made it
// untestable outside a DOM and unusable on a server. Capturing them in a struct
// keeps the detection logic pure while allowing the WebAssembly layer
// (cmd/wasm) to reproduce the original behaviour exactly.
//
// Field-by-field correspondence with the TypeScript source:
//
//	UserAgent              window.navigator.userAgent.toLowerCase()
//	Platform               navigator.platform
//	MaxTouchPoints         navigator.maxTouchPoints
//	HasScreenOrientation   Boolean(screen.orientation)
//	ScreenOrientationType  screen.orientation.type
//	HasOnOrientationChange Object.prototype.hasOwnProperty.call(window, 'onorientationchange')
//	HasWindowOrientation   Object.prototype.hasOwnProperty.call(window, 'orientation')
//	WindowOrientation      window.orientation
//	InnerWidth             window.innerWidth
//	InnerHeight            window.innerHeight
//	HasCordova             Boolean(window.cordova)
//	LocationProtocol       location.protocol
//	HasProcessObject       typeof window.process === 'object'
type Env struct {
	// UserAgent is the client user agent string. It is lower-cased by New and
	// SetEnv, mirroring the single `.toLowerCase()` call in the TypeScript
	// source, so that detection can use substring search instead of regexes.
	UserAgent string

	// Platform is navigator.platform. It is NOT lower-cased: the original
	// compares it against the exact string "MacIntel".
	Platform string

	// MaxTouchPoints is navigator.maxTouchPoints, used together with Platform
	// to detect an iPadOS 13+ device requesting the desktop site.
	MaxTouchPoints int

	// HasScreenOrientation reports whether screen.orientation was truthy.
	HasScreenOrientation bool

	// ScreenOrientationType is screen.orientation.type, e.g.
	// "portrait-primary" or "landscape-secondary".
	ScreenOrientationType string

	// HasOnOrientationChange mirrors
	// Object.prototype.hasOwnProperty.call(window, 'onorientationchange').
	HasOnOrientationChange bool

	// HasWindowOrientation mirrors
	// Object.prototype.hasOwnProperty.call(window, 'orientation').
	HasWindowOrientation bool

	// WindowOrientation is window.orientation, in degrees (0, 90, -90, 180).
	WindowOrientation int

	// InnerWidth is window.innerWidth.
	InnerWidth float64

	// InnerHeight is window.innerHeight.
	InnerHeight float64

	// HasCordova reports whether window.cordova was truthy.
	HasCordova bool

	// LocationProtocol is location.protocol, including the trailing colon,
	// e.g. "file:" or "https:".
	LocationProtocol string

	// HasProcessObject reports whether `typeof window.process === 'object'`,
	// which the original uses as its NW.js (node-webkit) probe.
	HasProcessObject bool
}

// normalize returns a copy of e with the user agent lower-cased, mirroring the
// single `userAgent.toLowerCase()` performed by the TypeScript module.
func (e Env) normalize() Env {
	e.UserAgent = strings.ToLower(e.UserAgent)
	return e
}
