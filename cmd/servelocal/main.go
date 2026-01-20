package main

import (
	"log"
	"net/http"

	"mavgo-flight/sqliter"
)

func main() {
	cfg := sqliter.DefaultConfig()
	server := sqliter.NewServer(cfg)

	http.Handle("/", server)
	log.Println("Serving local sqlite files from sample_data at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
