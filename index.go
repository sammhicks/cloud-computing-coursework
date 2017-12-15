package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())

	defer cancelCtx()

	projectID, projectIDDeclared := os.LookupEnv("PROJECT_ID")

	if !projectIDDeclared {
		log.Println("Project ID not declared")
		return
	}

	googleLoginAppID, googleLoginAppIDDeclared := os.LookupEnv("GOOGLE_SIGN_IN_APP_ID")

	if !googleLoginAppIDDeclared {
		log.Println("Google Login App ID not declared")
		return
	}

	storageBucketName, storageBucketNameDeclared := os.LookupEnv("STORAGE_BUCKET")

	if !storageBucketNameDeclared {
		log.Println("Storage Bucket not declared")
		return
	}

	log.Println("Connecting to PubSub")
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Println("Error Connecting to PubSub:", err)
		return
	}

	log.Println("Connecting to FileStore")
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Println("Error connecting to FileStore:", err)
		return
	}

	log.Println("Creating bucket")
	storageBucket := storageClient.Bucket(storageBucketName)

	log.Println("Connecting to DataStore")
	datastoreClient, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.Println("Error connecting to DataStore:", err)
	}

	port, portDeclared := os.LookupEnv("PORT")

	if !portDeclared {
		port = "8080"
		log.Println("Port not declared, defaulting to", port)
	}

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt)

	mux := http.NewServeMux()

	mux.Handle(EventsHandler(ctx, googleLoginAppID, pubsubClient, storageBucketName, storageBucket, datastoreClient))
	mux.Handle(UploadHandler(projectID, googleLoginAppID, pubsubClient, storageBucketName, storageBucket, datastoreClient))
	mux.Handle(CleanCookiesHandler(datastoreClient))
	mux.Handle(HealthHandler())
	mux.Handle("/", http.FileServer(http.Dir("static")))

	s := &http.Server{
		Addr:    ":" + port,
		Handler: mux}

	go func() {
		log.Println("Creating server...")

		defer cancelCtx()

		if err := s.ListenAndServe(); err == http.ErrServerClosed {
			log.Println("Server closed")
		} else if err != nil {
			log.Println("Error listening:", err)
			stop <- os.Interrupt
		}
	}()

	<-stop

	signal.Stop(stop)

	log.Println("Shutting down...")

	if err := s.Shutdown(context.Background()); err != nil {
		log.Println("Error shutting down server:", err)
	}
}
