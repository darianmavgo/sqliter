package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/darianmavgo/sqliter/sqliter"
)

func main() {
	// Setup a sample DB path
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "sample_data")

	// Config
	cfg := sqliter.DefaultConfig()
	cfg.ServeFolder = sampleDir
	cfg.BaseURL = "/custom"
	cfg.Verbose = true

	// Create Server
	srv := sqliter.NewServer(cfg)

	// Mount
	mux := http.NewServeMux()

	// IMPORTANT: We must StripPrefix so the server sees paths relative to its root
	mux.Handle("/custom/", http.StripPrefix("/custom", srv))

	// Also serve a root handler to show we are distinct
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Root handler. Go to <a href='/custom/'>/custom/</a>"))
	})

	addr := ":8080"
	fmt.Printf("Listening on http://localhost%s\n", addr)
	fmt.Printf("Test URL: http://localhost%s/custom/\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
