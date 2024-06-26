package main

import (
	"log"
	"net/http"
	"ws/internal/handlers"
)

func main() {

	mux := routes()
	log.Println("Starting channel listener")
	go handlers.ListenToWsChannel()

	log.Println("Starting web server on port 5123")
	_ = http.ListenAndServe(":5123", mux)
	//_ = http.ListenAndServe(":8080", mux)

}
