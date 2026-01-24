package main

import (
	"log"
	"net/http"

	"github.com/darianmavgo/sqliter/server"
	"github.com/darianmavgo/sqliter/sqliter"
)

func main() {
	cfg, err := sqliter.LoadConfig("config.hcl")
	if err != nil {
		log.Fatal(err)
	}
	srv := server.NewServer(cfg)

	// Serve static files for the theme
	http.Handle("/style1/", http.StripPrefix("/style1/", http.FileServer(http.Dir("themes/style1"))))
	http.Handle("/extensions/", http.StripPrefix("/extensions/", http.FileServer(http.Dir("extensions"))))
	http.Handle("/cssjs/", http.StripPrefix("/cssjs/", http.FileServer(http.Dir("cssjs"))))

	http.Handle("/", srv)
	log.Printf("Serving local sqlite files from %s at http://localhost:%s\n", cfg.DataDir, cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
