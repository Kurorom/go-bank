package main

import (
	"log"
)

func main() {
	store, err := newPostgresStore()
	if err != nil {
		log.Fatal(err)
	}
	if err := store.init(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	server := NewAPIServer(":3000", store)
	server.Run()
}
