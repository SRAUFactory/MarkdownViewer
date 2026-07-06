package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	home := os.Getenv("MARK_DOWN_HOME")
	if home == "" {
		home = "."
	}

	appHandler, err := NewAppHandler(home)
	if err != nil {
		log.Fatalf("Failed to initialize app handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", appHandler.Index)
	mux.HandleFunc("GET /{name...}", appHandler.View)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
