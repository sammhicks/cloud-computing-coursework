package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"

	"google.golang.org/appengine"
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
	storageBucket := storageClient.Bucket(storageBucketName)

	http.Handle(WebsocketHandler(googleLoginAppID, pubsubClient, storageBucketName, storageBucket))
	http.Handle(TestHandler(googleLoginAppID, pubsubClient, storageBucketName, storageBucket))
	http.Handle("/", http.FileServer(http.Dir("static")))

	log.Println("Starting AppEngine")

	appengine.Main()
}
