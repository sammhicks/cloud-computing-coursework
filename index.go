package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port, portDeclared := os.LookupEnv("PORT")

	if !portDeclared {
		port = "8080"
		log.Println("Port not declared, defaulting to", port)
	}

	mux := http.NewServeMux()

	mux.Handle(TransmitHandler())
	mux.Handle(EventsHandler())
	mux.Handle("/", http.FileServer(http.Dir("static")))

	s := &http.Server{
		Addr:    ":" + port,
		Handler: mux}

	log.Fatal(s.ListenAndServe())
}
