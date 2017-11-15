package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const transmitPath = "/transmit"

type transmitHandler struct {
}

func (h *transmitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		ctx := r.Context()

		storageClient, err := storage.NewClient(ctx)
		if err != nil {
			log.Println("Error creating storage client:", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		defer storageClient.Close()

		bkt := storageClient.Bucket("cloud-computing-coursework.appspot.com")

		obj := bkt.Object(fmt.Sprintf("test/%d", time.Now().Unix()))

		objWriter := obj.NewWriter(ctx)

		if _, err := io.Copy(objWriter, r.Body); err != nil {
			log.Println("Error streaming data:", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		if err := objWriter.Close(); err != nil {
			log.Println("Error closing writer:", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		projectID, projectIDDeclared := os.LookupEnv("project")

		if !projectIDDeclared {
			log.Println("Project ID not declared")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		pubsubClient, err := pubsub.NewClient(ctx, projectID)
		if err != nil {
			log.Println("Failed to create client:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		topic := pubsubClient.TopicInProject("test", projectID)

		topic.Publish(ctx, &pubsub.Message{Data: []byte("new message")})

		fmt.Fprintln(w, "Successfuly transmitted snippet")
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//TransmitHandler handles the transmission of a new snippet
func TransmitHandler() (string, http.Handler) { return transmitPath, &transmitHandler{} }
