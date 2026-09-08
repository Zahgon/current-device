/**
 * Browser loader for the Go/WebAssembly build of current-device.
 *
 * The upstream TypeScript package shipped a self-executing IIFE bundle, so
 * `window.device` existed the moment the <script> tag finished parsing. A
 * WebAssembly module cannot be instantiated synchronously, so that guarantee is
 * replaced by a promise: `await load()` resolves to the exact same `device`
 * object, with the same side effects already applied to <html>.
 *
 * Requires Go's `wasm_exec.js` to have been loaded first (it defines `Go`).
 * Copy it from `$(go env GOROOT)/lib/wasm/wasm_exec.js`, or run `make wasm`.
 */

const DEFAULT_WASM_URL = new URL('./current-device.wasm', import.meta.url)

let pending = null

function resolveGoConstructor(provided) {
  const Ctor = provided ?? globalThis.Go
  if (typeof Ctor !== 'function') {
    throw new Error(
      'current-device: `Go` is not defined. Load wasm_exec.js before calling load().'
    )
  }
  return Ctor
}

async function instantiate(url, importObject) {
  const response = fetch(url)
  if (typeof WebAssembly.instantiateStreaming === 'function') {
    try {
      return await WebAssembly.instantiateStreaming(response, importObject)
    } catch (error) {
      // Servers that mislabel .wasm as application/octet-stream reject the
      // streaming path; fall through to the buffered form rather than fail.
      if (!(error instanceof TypeError)) {
        throw error
      }
    }
  }
  const buffer = await (await fetch(url)).arrayBuffer()
  return WebAssembly.instantiate(buffer, importObject)
}

async function waitForGlobal(attempts = 50) {
  for (let i = 0; i < attempts; i += 1) {
    if (globalThis.device !== undefined) {
      return globalThis.device
    }
    await new Promise((resolve) => setTimeout(resolve, 0))
  }
  throw new Error('current-device: the wasm module did not publish `window.device`.')
}

/**
 * Instantiate the module and resolve with the global `device` object.
 *
 * Repeated calls return the same promise; the Go module installs itself once,
 * mirroring the single-evaluation semantics of an ES module import.
 *
 * @param {string|URL} [wasmURL] Location of `current-device.wasm`.
 * @param {{ Go?: Function }} [options]
 * @returns {Promise<object>} the `device` API.
 */
export function load(wasmURL = DEFAULT_WASM_URL, options = {}) {
  if (pending !== null) {
    return pending
  }

  pending = (async () => {
    const Ctor = resolveGoConstructor(options.Go)
    const go = new Ctor()
    const { instance } = await instantiate(wasmURL, go.importObject)

    // `main()` parks forever to keep the exported js.Func handles alive, so the
    // promise returned by run() never settles. Awaiting it would deadlock.
    go.run(instance)

    return waitForGlobal()
  })()

  return pending
}

export default { load }

globalThis.CurrentDevice = { load }
