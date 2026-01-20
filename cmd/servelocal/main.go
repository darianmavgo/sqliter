package main

import (
	"log"
	"net/http"

	"github.com/darianmavgo/sqliter/pkg/server"
	"github.com/darianmavgo/sqliter/sqliter"
)

func main() {
	cfg := sqliter.DefaultConfig()
	srv := server.NewServer(cfg)

	http.Handle("/", srv)
	log.Println("Serving local sqlite files from sample_data at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
