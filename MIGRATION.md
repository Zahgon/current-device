# Migration: current-device (TypeScript) → current-device-go

This document records how the port was done, what was deliberately preserved,
what necessarily changed, and how equivalence is proven.

The source of truth was `src/index.ts` from
[matthewhudson/current-device][repo] v2.1.0 — a single 437-line browser module.

[repo]: https://github.com/matthewhudson/current-device

## 1. Architecture

The original is a side-effectful singleton. At import time it reads browser
globals, mutates `document.documentElement.className`, registers a `resize` or
`orientationchange` listener, and assigns itself to `window.device`. Roughly 40%
of its observable behaviour *is* DOM interaction, so a pure-Go library alone
would not have been a complete port.

The port is therefore two layers:

| Layer | Package | Build | Responsibility |
| --- | --- | --- | --- |
| Core | `device` (repo root) | any GOOS | All detection. Pure functions of an injected `Env`. No globals, no `init`, no I/O. |
| Browser | `cmd/wasm` | `GOOS=js GOARCH=wasm` | Reads real browser globals into an `Env`, performs every side effect, exports `window.device`. |

`cmd/wasm/stub.go` carries a `!js || !wasm` build tag so that `go build ./...`
still succeeds on ordinary platforms, where `syscall/js` is unavailable.

`web/current-device.js` replaces the tsup IIFE bundle (`dist/index.global.js`)
that the demo page and the unpkg/jsdelivr CDN entries used to consume.

## 2. Symbol mapping

All 30 public members map 1:1. Predicate names are unchanged in the WebAssembly
export (`device.androidPhone()`); the Go core uses Go naming
(`d.AndroidPhone()`). The full table is in [README.md](README.md#api).

Three TypeScript *properties* become Go *methods*, because Go has no property
accessors:

| TypeScript | Go |
| --- | --- |
| `device.type` | `d.Type() DeviceType` |
| `device.os` | `d.OS() DeviceOS` |
| `device.orientation` | `d.Orientation() DeviceOrientation` |

Their caching semantics are preserved: `Type` and `OS` are computed once at
construction, `Orientation` is recomputed by every `HandleOrientation` call. The
WebAssembly layer re-exports all three as plain string properties, so the
JavaScript-visible API is unchanged.

Types added for safety, with the same string values as the TypeScript string
unions: `DeviceType`, `DeviceOS`, `DeviceOrientation`, plus named constants
(`TypeMobile`, `OSIOS`, `OrientationPortrait`, …).

## 3. Deliberate divergences

Everything below is a difference. There are no others.

### 3.1 `Env` injection replaces module-scope globals

The TypeScript reads `window`, `navigator`, `screen`, `location` and `document`
directly at module scope. The Go core takes an `Env` struct. This is what makes
the package testable without a DOM and usable on a server, and it is the only
structural change to the design.

The WebAssembly layer restores the original behaviour by populating `Env` from
the real globals — and re-reads it before every predicate call, so
`portrait()`/`landscape()` observe the live viewport exactly as the original's
direct `window.innerWidth` reads did.

### 3.2 `HasClass` on an uncompilable pattern

The original implements `hasClass` as
`className.match(new RegExp(className, 'i'))` — the class string is compiled as a
*regular expression*. Go's `regexp` returns an error where JavaScript's `RegExp`
throws. Rather than panic, `HasClass` treats an uncompilable pattern as absent.

This is unreachable in practice: the library only ever passes its own literal
class names, none of which contain regex metacharacters. It is covered by
`TestHasClassWithInvalidPattern`.

Note that this regex-based implementation — not word matching — is why the
multi-token strings (`"ios ipad tablet"`) work as single patterns. That is
preserved.

### 3.3 `window.orientation` is an `int`

JavaScript numbers are float64. `window.orientation` is only ever `0`, `90`,
`-90` or `180`, so `Env.WindowOrientation` is an `int`. The
`Math.abs(orientation) !== 90` comparison is reproduced exactly.

Viewport dimensions remain `float64`, which is required to reproduce the
`Infinity` and `NaN` results of the original's division — see §4.

### 3.4 `NoConflict` in the core package

`noConflict` restores `window.device`, which is meaningless without a browser. In
the core package `NoConflict()` is a no-op returning the receiver, preserving the
chainable signature. The WebAssembly layer implements the real behaviour,
including restoring `undefined` (not `null`) when there was no previous value.

### 3.5 Asynchronous browser initialisation

WebAssembly cannot be instantiated synchronously, so `window.device` appears when
`load()` resolves rather than when the `<script>` tag finishes parsing. This is a
platform constraint, not a design choice. The ordering of the side effects
*within* initialisation is unchanged: export the API → set `window.device` →
stamp classes → register the listener → run `handleOrientation()` once.

### 3.6 Concurrency guarding

`Device` carries a `sync.RWMutex`, which has no counterpart in single-threaded
JavaScript. Orientation callbacks are copied out from under the lock before being
invoked, so a callback may re-enter the `Device` without deadlocking.

### 3.7 `nodeWebkit()` ignores Go's `process` polyfill

The source tests `typeof window.process === 'object'`. Transcribing that literally
produces a **wrong** answer under WebAssembly: Go's own `wasm_exec.js` executes

```js
if (!globalThis.process) { globalThis.process = { getuid() {...}, cwd() {...}, ... } }
```

at script-parse time, so a stub `process` exists in every browser before the
binary starts. A literal port therefore reports `nodeWebkit() === true` in plain
Chrome, where the TypeScript reports `false`.

That is not cosmetic. `macos` and `windows` are evaluated before `nodeWebkit` in
the class cascade, so the fault is invisible on the two most common desktops — but
a Linux desktop or a smart TV would be stamped `node-webkit` instead of `desktop`
or `television`.

The WASM layer therefore additionally requires `process.versions` to be an object.
Every genuine Node and NW.js `process` carries `versions`; Go's polyfill does not.
The result matches the TypeScript in both real-world cases:

| Runtime | `window.process` | TypeScript | Go port |
| --- | --- | --- | --- |
| Plain browser | Go polyfill (no `versions`) | `false` | `false` |
| NW.js | real Node process | `true` | `true` |

The core `device` package is unaffected: it consumes `Env.HasProcessObject`
verbatim, so server-side callers keep the literal semantics. Locked by
`TestNodeWebkitDetection` and `TestGoProcessPolyfillDoesNotHijackTheClassCascade`
in `cmd/wasm/main_test.go`.

### 3.8 Type names keep the TypeScript spelling

`DeviceType`, `DeviceOS` and `DeviceOrientation` stutter when qualified
(`device.DeviceType`), and `revive` flags them. Idiomatic Go would name them
`Type`, `OS` and `Orientation`.

They are kept as-is because they are the 1:1 counterparts of the TypeScript
package's exported `DeviceType` / `DeviceOs` / `DeviceOrientation` types, and
that correspondence is the published migration contract in section 2. The
warning is suppressed by a narrow, path- and text-scoped exclusion in
`.golangci.yml` rather than by disabling `revive`. No behavioural impact.

## 4. Preserved quirks

Each of these looks like a bug. Each is reproduced, and each is locked by both a
unit test and a differential scenario. Changing any of them would be a breaking
change for a package with ~25k weekly downloads.

| Behaviour | Why it happens | Locked by |
| --- | --- | --- |
| Every iOS device reports `os == "ios"`; `"iphone"`, `"ipad"`, `"ipod"` are unreachable | `"ios"` precedes them in the OS resolution array | `TestIOSShadowsSpecificOSLabels` |
| A smart TV reports `type == "desktop"` | `desktop()` is `!tablet() && !mobile()`, and `television()` is not consulted for type | `class:television` |
| NW.js outranks television in the cascade, and adds no OS/form-factor class | `nodeWebkit` is tested before `television` in the if/else chain | `TestNodeWebkitPrecedesTelevision`, `class:nodeWebkit-beats-tv` |
| An iOS device that is neither iPad, iPhone nor iPod adds no class | The inner cascade has no `else` | `TestIOSWithoutSpecificDeviceAddsNothing` |
| A square viewport yields `orientation == "unknown"` — but still gets the `portrait` class | Probes are `ratio > 1` and `ratio < 1`; `handleOrientation` falls through to the portrait branch | `TestSquareViewportBothProbesFalse`, `orient:square` |
| A zero-width viewport reports portrait | `800/0` is `+Inf`, and `+Inf > 1` | `TestZeroViewportMatchesJavaScriptSemantics`, `orient:zero-width` |
| `NewFromUserAgent` reports `orientation == "unknown"` | `0/0` is `NaN`, and every `NaN` comparison is false | `TestUADetection` |
| The first class added to an empty string keeps a leading space (`" desktop"`) | `addClass` trims the *subject*, then appends `" " + className` | `TestAddClass` |
| …but a second `addClass` removes it (`"ios iphone mobile cordova"`) | the same trim runs again on the now-non-empty subject | `TestCordovaClassLosesLeadingSpace`, `class:cordova` |
| An iPod touch takes the *iPhone* cascade branch | real iPod user agents contain `"CPU iPhone OS"`, so `iphone()` is true and is tested first | `TestIPodClassQuirk`, `class:ipod-only` |
| Firefox for Android is not detected as Firefox OS | the needles are `"(mobile"` and `"(tablet"` with a leading parenthesis; Firefox for Android emits `"; Mobile;"` | `TestFxOS` |
| `removeClass` only removes the first occurrence, and only when space-prefixed | `String.replace` with a string argument replaces once | `TestRemoveClass` |
| `MeeGo` adds `"meego mobile"`, not `"meego"` as the README claims | README drift; the code appends both | `TestClassNames` |

Two upstream **documentation** errors were also found and are corrected in this
port's README (the code behaviour is unchanged): the class table omits the
HarmonyOS and NW.js rows, and lists MeeGo as `meego`.

## 5. A note on the source's bare globals

`src/index.ts` reads `window.navigator.userAgent` (line 76) but elsewhere uses
the *bare* globals `navigator` (line 154), `location` (line 215) and `screen`
(lines ~260, ~276). In a browser these are identical, since they resolve to the
same objects via the global scope. They are noted here because they matter to
anyone reproducing the oracle harness outside a browser — under Node they must be
patched onto `globalThis`, and `navigator` requires `Object.defineProperty`
because Node's is getter-only.

The Go WebAssembly layer reads all of them from the `window` object it is given,
which is equivalent in a browser and cleaner to test.

## 6. Equivalence proof

### 6.1 Method

Node 26 executes TypeScript directly (type stripping), so `src/index.ts` was run
*as the original*, not transpiled or reimplemented. `verification/oracle-harness.ts`
constructs a synthetic `window` per scenario, patches `globalThis`, then performs
a cache-busted dynamic `import('./index.ts?v=N')` to obtain a fresh
side-effectful module instance. For each scenario it records the environment
inputs and the full observable output: `os`, `type`, `orientation`, the resulting
`className`, the chosen orientation event name, and all 27 predicate results.

The result is `testdata/typescript_oracle.json` — 53 scenarios.
`TestDifferentialAgainstTypeScript` replays every one against the Go
implementation and asserts field-by-field equality. Regenerate with
`make oracle`.

`TestOracleExercisesBothOutcomes` asserts that every one of the 27 predicates is
observed both `true` and `false` somewhere in the corpus, which prevents the
suite from being satisfied by a port that returns constants.

### 6.2 Scenario coverage

All 23 user agent fixtures from the upstream `tests/ua-strings.ts` are carried
over verbatim, with their expected `os`/`type` and per-method assertions:

| Scenario | os | type |
| --- | --- | --- |
| iPhone Safari (iOS 18) | `ios` | `mobile` |
| iPhone Chrome (iOS 18) | `ios` | `mobile` |
| iPhone (older iOS 15) | `ios` | `mobile` |
| iPad Safari (mobile UA, iPadOS 17) | `ios` | `tablet` |
| iPad Safari (desktop mode, iPadOS 13+) | `ios` | `tablet` |
| iPod Touch Safari | `ios` | `mobile` |
| Android Chrome phone | `android` | `mobile` |
| Samsung Browser phone | `android` | `mobile` |
| Android Chrome (older, specific device) | `android` | `mobile` |
| Android tablet Chrome | `android` | `tablet` |
| macOS Safari | `macos` | `desktop` |
| macOS Chrome | `macos` | `desktop` |
| Windows Chrome | `windows` | `desktop` |
| Windows Edge | `windows` | `desktop` |
| Windows Firefox | `windows` | `desktop` |
| Linux Chrome | `unknown` | `desktop` |
| Linux Firefox | `unknown` | `desktop` |
| Smart TV (generic) | `television` | `desktop` |
| Roku | `television` | `desktop` |
| Apple TV | `television` | `desktop` |
| HarmonyOS phone (Huawei Browser) | `harmonyos` | `mobile` |
| HarmonyOS tablet (Huawei Browser) | `harmonyos` | `tablet` |
| Windows Phone | `windows` | `mobile` |

The upstream fixtures leave several predicates permanently false, so 30 further
scenarios were added to exercise them:

- **13 user agents** covering BlackBerry (phone, tablet, PlayBook, BB10),
  Firefox OS (phone, tablet), MeeGo, Windows tablet and phone, television, an
  iPod-only user agent, HarmonyOS and Linux.
- **9 orientation configurations**: both `screen.orientation` branches, the
  legacy iOS `window.orientation` angles (0, 90, −90, 180), the viewport-ratio
  fallback, a square viewport and a zero-width viewport.
- **8 class cascade cases**: Cordova, NW.js, NW.js-beats-television, television,
  pre-existing classes on `<html>`, macOS, iPod-only and HarmonyOS tablet.

### 6.3 Results

| Check | Result |
| --- | --- |
| Differential scenarios | 53 / 53 pass |
| Upstream UA fixtures | 23 / 23 pass |
| Core package coverage | 100.0% of statements |
| `cmd/wasm` coverage | 94.8% (remainder is `main`, which parks forever) |
| `gofmt -l .` | clean |
| `go vet ./...` | clean |
| Fuzzing | no panics; resolved values always declared constants |

**Zero behavioural differences** were found for all 53 scenarios and all
README-documented class combinations.

## 7. For consumers

### JavaScript / browser

The API is unchanged; only acquisition differs.

```diff
-<script src="https://unpkg.com/current-device"></script>
-<script>
-  console.log(device.type)
-</script>
+<script src="wasm_exec.js"></script>
+<script type="module">
+  import { load } from './current-device.js'
+  const device = await load('./current-device.wasm')
+  console.log(device.type)
+</script>
```

Everything after that line — `device.mobile()`, `device.os`,
`device.onChangeOrientation(cb)`, `device.noConflict()`, the `<html>` classes —
behaves identically.

### Go

New audience, new entry point:

```go
d := device.NewFromUserAgent(r.UserAgent())
```

The main adjustment from the JavaScript mental model: `type`, `os` and
`orientation` are method calls, and orientation requires viewport data that a
user agent string does not carry.

## 8. Toolchain mapping

| TypeScript | Go |
| --- | --- |
| `pnpm build` (tsup → CJS/ESM/IIFE/`.d.ts`) | `make wasm` (wasm binary + `wasm_exec.js` + loader) |
| `pnpm test` (vitest + jsdom) | `make test` (`go test -race -cover`) |
| — | `make test-wasm` (`GOOS=js GOARCH=wasm go test`, executed under Node) |
| `pnpm typecheck` (`tsc --noEmit`) | `go vet` + `golangci-lint` |
| Changesets | git tags (Go modules are tag-versioned) |
| `.github/workflows/ci.yml` | `.github/workflows/ci.yml` (matrix: 3 OSes × 3 Go versions, plus wasm, lint and differential jobs) |

One upstream testing hazard disappears: under jsdom, `window.process` exists, so
`device.nodeWebkit()` was always true and the `node-webkit` class always won the
cascade, which the upstream `CLAUDE.md` documents as a trap. Because the Go core
takes an explicit `Env`, `HasProcessObject` is false unless a test sets it.
