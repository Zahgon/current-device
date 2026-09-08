# current-device-go

[![CI](https://github.com/matthewhudson/current-device-go/actions/workflows/ci.yml/badge.svg)](https://github.com/matthewhudson/current-device-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/matthewhudson/current-device-go.svg)](https://pkg.go.dev/github.com/matthewhudson/current-device-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/matthewhudson/current-device-go)](https://goreportcard.com/report/github.com/matthewhudson/current-device-go)

Device detection for Go — operating system, form factor and orientation.

A port of the npm package [current-device][npm], intended to be behaviourally
identical to it. Detection logic is transcribed, not reimplemented: the same
substring needles, the same evaluation order, the same edge cases. Equivalence is
[verified against the original TypeScript](#equivalence-testing) rather than
assumed.

[npm]: https://www.npmjs.com/package/current-device

## Two layers

The original is a browser module with import-time side effects. Go cannot express
that directly, so the port is split:

| Layer | Location | Role |
| --- | --- | --- |
| Core | this package | Pure detection. No globals, no `init`, no I/O. Runs anywhere, including on a server. |
| Browser | `cmd/wasm` | `GOOS=js GOARCH=wasm` bindings. Reads real browser globals, stamps `<html>` classes, listens for orientation changes, publishes `window.device`. |

## Installation

```sh
go get github.com/matthewhudson/current-device-go
```

Requires Go 1.22 or newer. The core package has no dependencies outside the
standard library.

## Usage

### Server-side

```go
import device "github.com/matthewhudson/current-device-go"

func handler(w http.ResponseWriter, r *http.Request) {
	d := device.NewFromUserAgent(r.UserAgent())

	if d.Mobile() {
		// ...
	}

	fmt.Println(d.OS())   // "ios", "android", "macos", ...
	fmt.Println(d.Type()) // "mobile", "tablet", "desktop"
}
```

A user agent alone cannot reveal orientation, so `d.Orientation()` reports
`"unknown"` unless you supply viewport information.

### With a full environment

`NewFromUserAgent` is shorthand for `New` with everything else zeroed. Supply an
[`Env`](https://pkg.go.dev/github.com/matthewhudson/current-device-go#Env) to
reproduce the full browser signal set — including the `MacIntel` +
`MaxTouchPoints` check that distinguishes an iPad in desktop mode from a Mac:

```go
d := device.New(device.Env{
	UserAgent:      ua,
	Platform:       "MacIntel",
	MaxTouchPoints: 5,
	InnerWidth:     834,
	InnerHeight:    1194,
})

d.IPad()          // true
d.Orientation()   // "portrait"
```

### Browser (WebAssembly)

```sh
make wasm   # -> dist/current-device.wasm, wasm_exec.js, current-device.js
```

```html
<script src="wasm_exec.js"></script>
<script type="module">
  import { load } from './current-device.js'

  const device = await load('./current-device.wasm')

  if (device.mobile()) {
    // ...
  }

  console.log(device.type, device.os, device.orientation)
</script>
```

The resolved `device` object has the same 30 members as the npm package, with the
same camelCase names. The one unavoidable difference: WebAssembly cannot be
instantiated synchronously, so `window.device` is published when the returned
promise resolves rather than when the `<script>` tag finishes parsing.

Run `make serve` for a local demo at <http://localhost:8080> — WebAssembly cannot
be fetched over `file://`.

## Conditional CSS

The WebAssembly layer stamps these classes onto `<html>`, exactly as the original
does. The core package exposes the same cascade as a pure function,
`d.ClassNames(current)`, so it can also be used for server-side rendering.

### Device classes

| Device | CSS classes |
| --- | --- |
| iPad | `ios ipad tablet` |
| iPhone | `ios iphone mobile` |
| iPod | `ios ipod mobile` |
| Mac | `macos desktop` |
| HarmonyOS phone | `harmonyos mobile` |
| HarmonyOS tablet | `harmonyos tablet` |
| Android phone | `android mobile` |
| Android tablet | `android tablet` |
| BlackBerry phone | `blackberry mobile` |
| BlackBerry tablet | `blackberry tablet` |
| Windows phone | `windows mobile` |
| Windows tablet | `windows tablet` |
| Windows desktop | `windows desktop` |
| Firefox OS phone | `fxos mobile` |
| Firefox OS tablet | `fxos tablet` |
| MeeGo | `meego mobile` |
| NW.js | `node-webkit` |
| Television | `television` |
| Desktop | `desktop` |

`cordova` is appended independently, so it combines with any row above.

> The upstream README omits the HarmonyOS and NW.js rows and lists MeeGo as
> `meego` rather than `meego mobile`. The table here reflects what the code
> actually does, which is what this port reproduces.

### Orientation classes

| Orientation | CSS class |
| --- | --- |
| Landscape | `landscape` |
| Portrait | `portrait` |

## API

Thirty members, mapped 1:1 from the TypeScript interface. Predicates are grouped
below; see the [reference docs][godoc] for details.

[godoc]: https://pkg.go.dev/github.com/matthewhudson/current-device-go

| TypeScript | Go |
| --- | --- |
| `device.macos()` | `d.MacOS()` |
| `device.ios()` | `d.IOS()` |
| `device.iphone()` | `d.IPhone()` |
| `device.ipod()` | `d.IPod()` |
| `device.ipad()` | `d.IPad()` |
| `device.android()` | `d.Android()` |
| `device.androidPhone()` | `d.AndroidPhone()` |
| `device.androidTablet()` | `d.AndroidTablet()` |
| `device.blackberry()` | `d.Blackberry()` |
| `device.blackberryPhone()` | `d.BlackberryPhone()` |
| `device.blackberryTablet()` | `d.BlackberryTablet()` |
| `device.windows()` | `d.Windows()` |
| `device.windowsPhone()` | `d.WindowsPhone()` |
| `device.windowsTablet()` | `d.WindowsTablet()` |
| `device.fxos()` | `d.FxOS()` |
| `device.fxosPhone()` | `d.FxOSPhone()` |
| `device.fxosTablet()` | `d.FxOSTablet()` |
| `device.meego()` | `d.MeeGo()` |
| `device.harmonyos()` | `d.HarmonyOS()` |
| `device.television()` | `d.Television()` |
| `device.cordova()` | `d.Cordova()` |
| `device.nodeWebkit()` | `d.NodeWebkit()` |
| `device.mobile()` | `d.Mobile()` |
| `device.tablet()` | `d.Tablet()` |
| `device.desktop()` | `d.Desktop()` |
| `device.portrait()` | `d.Portrait()` |
| `device.landscape()` | `d.Landscape()` |
| `device.onChangeOrientation(cb)` | `d.OnChangeOrientation(cb)` |
| `device.noConflict()` | `d.NoConflict()` |
| `device.type` | `d.Type()` |
| `device.os` | `d.OS()` |
| `device.orientation` | `d.Orientation()` |

Full details, including the rationale for each preserved quirk, are in
[MIGRATION.md](MIGRATION.md).

### Orientation callbacks

```go
d.OnChangeOrientation(func(o device.DeviceOrientation) {
	log.Println(o) // "portrait" or "landscape"
})
```

Callbacks fire from `d.HandleOrientation(className)`, which the WebAssembly layer
invokes on every `orientationchange` (or `resize`) event. As in the original, one
`HandleOrientation` call happens at startup *before* user callbacks can be
registered, so a callback only observes subsequent changes.

## Behaviour worth knowing

These are upstream behaviours, reproduced deliberately. Each is locked by a test.

- **iOS shadows the specific labels.** `"ios"` precedes `"iphone"`, `"ipad"` and
  `"ipod"` in the OS resolution order, so those three values are unreachable —
  every iOS device reports `OS() == "ios"`. The predicates still work.
- **Televisions are desktops.** `Desktop()` is `!Tablet() && !Mobile()`, so a
  smart TV reports `OS() == "television"` and `Type() == "desktop"`.
- **NW.js outranks television** in the class cascade, and adds no OS or form
  factor class.
- **A square viewport is `"unknown"`.** The portrait probe is `height/width > 1`
  and the landscape probe is `< 1`; when they are equal, both are false.
- **`AddClass` trims first.** Appending to `""` yields `" desktop"`, with a
  leading space. Consumers split on whitespace, so this is harmless — and
  changing it would alter the emitted DOM.
- **`iphone` beats `ipod`.** Real iPod touch user agents contain
  `"CPU iPhone OS"`, so `IPhone()` is true and the cascade takes the iPhone
  branch.
- **`fxos` needles include punctuation.** They are `"(mobile"`, `"(tablet"` and
  `" rv:"`. Firefox for Android emits `"; Mobile;"`, so it is correctly *not*
  detected as Firefox OS.

## Equivalence testing

Three layers of verification, all run in CI:

1. **Differential tests.** `testdata/typescript_oracle.json` records 53 scenarios
   produced by executing the original `src/index.ts` under Node — 23 user agent
   fixtures carried over verbatim from the upstream suite, plus extra user
   agents, orientation configurations and class cascade cases.
   `TestDifferentialAgainstTypeScript` replays each and compares the resolved OS,
   type, orientation, resulting className, orientation event name and all 27
   predicates. Regenerate with `make oracle`.
2. **Unit tests.** Every predicate, every cascade branch, every orientation code
   path. The core package is at **100% statement coverage**; the WebAssembly
   bindings are at 94.8% (the remainder is `main`, which parks forever).
3. **Fuzzing.** `FuzzDetectionNeverPanics` asserts no input panics and that
   resolved values are always declared constants.

`TestOracleExercisesBothOutcomes` additionally asserts that every predicate is
observed both true *and* false across the corpus, so the suite cannot be
satisfied by a port that returns a constant.

```sh
make check      # fmt, vet, lint, tests, wasm tests, builds
make test       # core tests with race detector and coverage
make test-wasm  # syscall/js bindings, executed under Node
```

## Credits

Ported from [current-device][repo] by [Matthew Hudson][mh]. MIT licensed; the
original copyright is preserved in [LICENSE](LICENSE).

[repo]: https://github.com/matthewhudson/current-device
[mh]: https://github.com/matthewhudson
