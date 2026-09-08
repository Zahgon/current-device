//go:build !js || !wasm

// This stub keeps `go build ./...` working on non-WebAssembly toolchains, where
// syscall/js does not exist. Build the real entry point with GOOS=js GOARCH=wasm.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "current-device-go: build this command with GOOS=js GOARCH=wasm")
	os.Exit(1)
}
