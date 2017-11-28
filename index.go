package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

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

	log.Println("Connecting to pubsub")
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Println("Failed to create client:", err)
		return
	}

	log.Println("Connecting to storage")
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Println("Error creating storage client:", err)
		return
	}

	log.Println("Creating bucket")
	storageBucket := storageClient.Bucket("cloud-computing-coursework-storage")

	port, portDeclared := os.LookupEnv("PORT")

	if !portDeclared {
		port = "8080"
		log.Println("Port not declared, defaulting to", port)
	}

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt)

	mux := http.NewServeMux()

	mux.Handle(WebsocketHandler(pubsubClient, storageBucket))
	mux.Handle("/", http.FileServer(http.Dir("static")))

	s := &http.Server{
		Addr:    ":" + port,
		Handler: mux}

	go func() {
		log.Println("Creating server...")
		if err := s.ListenAndServe(); err != nil {
			log.Println("Error listening:", err)
		}
	}()

	<-stop

	signal.Stop(stop)

	log.Println("Shutting down...")

	s.Shutdown(context.Background())
}
