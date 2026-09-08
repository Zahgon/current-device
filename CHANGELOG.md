# Changelog

## 3.0.0

Complete port from TypeScript to Go. See [MIGRATION.md](MIGRATION.md).

The detection logic is unchanged — it was transcribed rather than
reimplemented, and equivalence with the TypeScript v2.1.0 implementation is
verified by 53 differential scenarios recorded from the original source. No
device is detected differently, and no CSS class output differs.

### Added

- `device` package: pure detection driven by an explicit `Env` snapshot, usable
  server-side via `device.NewFromUserAgent(r.UserAgent())`.
- `cmd/wasm`: `GOOS=js GOARCH=wasm` bindings reproducing every browser side
  effect — `<html>` classes, the orientation listener and `window.device`.
- `web/current-device.js`: promise-based loader replacing the IIFE bundle.
- Typed values: `DeviceType`, `DeviceOS`, `DeviceOrientation` with named
  constants, replacing the TypeScript string unions.
- Differential test suite replaying recorded output from the original
  `src/index.ts`.

### Changed

- **Breaking (JavaScript consumers):** `window.device` is published when
  `load()` resolves rather than synchronously at script-parse time. WebAssembly
  cannot be instantiated synchronously.
- **Breaking (JavaScript consumers):** distribution is a `.wasm` bundle plus
  loader instead of CJS/ESM/IIFE builds from npm.
- `type`, `os` and `orientation` are methods in Go (`d.Type()`, `d.OS()`,
  `d.Orientation()`). They remain plain string properties on the
  WebAssembly-exported object.
- `HasClass` treats an uncompilable pattern as absent where JavaScript's
  `RegExp` constructor would throw. Unreachable with the library's own class
  names.

### Preserved

Every upstream quirk is reproduced deliberately, including the `ios` OS label
shadowing `iphone`/`ipad`/`ipod`, televisions reporting `type: desktop`,
`node-webkit` outranking `television` in the class cascade, the leading space
from the first `addClass`, and the `unknown` orientation for a square viewport.
Each is documented in MIGRATION.md §4 and locked by tests.

### Documentation

- Corrected two errors carried in the upstream README's CSS class table: the
  HarmonyOS and NW.js rows were missing, and MeeGo emits `meego mobile` rather
  than `meego`. Behaviour is unchanged; only the documentation was wrong.

---

For the history of the TypeScript package (v1.x–v2.1.0), see the
[original repository](https://github.com/matthewhudson/current-device/blob/main/CHANGELOG.md).
