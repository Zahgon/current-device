// Package device detects the operating system, form factor and orientation of
// a client. It is a Go port of the npm package current-device, and is intended
// to be behaviourally identical to it.
//
// # Two layers
//
// The TypeScript original is a browser module with import-time side effects: it
// reads browser globals, stamps classes onto the <html> element, subscribes to
// orientation events and publishes itself as window.device. That design cannot
// be expressed as idiomatic Go, so the port is split in two.
//
// This package is the pure layer. It has no globals, no init function and no
// I/O; every detection method is a function of an [Env] snapshot supplied by the
// caller. That makes it usable server-side, where the only signal available is a
// request's user agent:
//
//	d := device.NewFromUserAgent(r.UserAgent())
//	if d.Mobile() {
//		// ...
//	}
//
// The command in cmd/wasm is the browser layer. Built with GOOS=js GOARCH=wasm,
// it populates an [Env] from the real browser globals, applies the class and
// event-listener side effects, and exports an object under window.device whose
// shape matches the original TypeScript API exactly. Run `make wasm` to build
// it together with the loader in web/current-device.js.
//
// # Fidelity
//
// Detection is deliberately a transcription rather than a reimplementation.
// User agent matching is plain substring search on a lowercased string, the
// evaluation orders in [Device.Type], [Device.OS] and [Device.ClassNames] are
// unchanged, and upstream quirks are reproduced rather than corrected. The ones
// most likely to surprise are documented on the methods that cause them:
//
//   - Every iOS device reports OS [OSIOS]; [OSIPhone], [OSIPad] and [OSIPod] are
//     unreachable, because "ios" precedes them in the resolution order.
//   - A smart TV reports OS [OSTelevision] but Type [TypeDesktop].
//   - A square viewport satisfies neither the portrait nor the landscape probe,
//     so [Device.Orientation] reports [OrientationUnknown].
//   - [AddClass] trims its subject before appending, so the first class added to
//     an empty string keeps a leading space.
//
// Equivalence is enforced, not asserted. testdata/typescript_oracle.json holds
// 53 scenarios recorded by executing the original src/index.ts under Node, and
// TestDifferentialAgainstTypeScript replays each one, comparing the resolved OS,
// type, orientation, resulting className, orientation event name and all 27
// predicates.
//
// # Concurrency
//
// A [Device] is safe for concurrent use by multiple goroutines.
package device
