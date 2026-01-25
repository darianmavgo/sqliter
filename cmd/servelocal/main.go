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

	log.Printf("Serving local sqlite files from %s at http://localhost:%s\n", cfg.DataDir, cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, srv))
}
