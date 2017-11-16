package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const transmitPath = "/transmit"

type transmitHandler struct {
	pubsubClient  *pubsub.Client
	storageBucket *storage.BucketHandle
}

func (h *transmitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		ctx := r.Context()

		keys, err := GoogleKeys()

		if err != nil {
			log.Println("Failed to fetch google public keys:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		userID, userEmail, err := VerifyToken(r.URL.Query().Get("token"), keys, "812818444262-dihtcq1cl07rrc4d3gs86obfs95dhe4i.apps.googleusercontent.com")

		if err != nil {
			log.Println("Auth Error:", err)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		topicName := fmt.Sprint("notifications-", userID)

		topic, err := h.pubsubClient.CreateTopic(ctx, topicName)

		if err != nil {
			topic = h.pubsubClient.Topic(topicName)
		}

		objectName := fmt.Sprintf("snippets/%x/%x", userID, time.Now().UnixNano())

		obj := h.storageBucket.Object(objectName)

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

		if err := obj.ACL().Set(ctx, storage.ACLEntity(fmt.Sprint("user-", userEmail)), storage.RoleReader); err != nil {

			log.Println("Error closing writer:", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		topic.Publish(ctx, &pubsub.Message{Data: []byte(objectName)})

		fmt.Fprintln(w, "Successfuly transmitted snippet")
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//TransmitHandler handles the transmission of a new snippet
func TransmitHandler(pubsubClient *pubsub.Client, storageBucket *storage.BucketHandle) (string, http.Handler) {
	return transmitPath, &transmitHandler{
		pubsubClient:  pubsubClient,
		storageBucket: storageBucket,
	}
}
