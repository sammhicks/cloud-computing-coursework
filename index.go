package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())

	defer cancelCtx()

	projectID, projectIDDeclared := os.LookupEnv("project")

	if !projectIDDeclared {
		log.Println("Project ID not declared")
		return
	}

	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Println("Failed to create client:", err)
		return
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Println("Error creating storage client:", err)
		return
	}

	storageBucket := storageClient.Bucket("cloud-computing-coursework.appspot.com")

	port, portDeclared := os.LookupEnv("PORT")

	if !portDeclared {
		port = "8080"
		log.Println("Port not declared, defaulting to", port)
	}

	mux := http.NewServeMux()

	mux.Handle(TransmitHandler(pubsubClient, storageBucket))
	mux.Handle(EventsHandler(pubsubClient, storageBucket))
	mux.Handle("/", http.FileServer(http.Dir("static")))

	s := &http.Server{
		Addr:    ":" + port,
		Handler: mux}

	log.Println(s.ListenAndServe())
}
