// Command serve hosts the current directory over HTTP for the WebAssembly demo.
//
// A static file server is required because browsers refuse to instantiate
// WebAssembly fetched over file://.
package main

import (
	"flag"
	"log"
	"mime"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dir := flag.String("dir", ".", "directory to serve")
	flag.Parse()

	// Without the correct MIME type WebAssembly.instantiateStreaming rejects
	// the response, and Go's built-in table does not always cover .wasm.
	if err := mime.AddExtensionType(".wasm", "application/wasm"); err != nil {
		log.Fatalf("serve: registering wasm mime type: %v", err)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: http.FileServer(http.Dir(*dir)),
	}

	log.Printf("serve: listening on %s", *addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
