package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const uploadPath = "/upload"

type uploadHandler struct {
	googleLoginAppID  string
	pubsubClient      *pubsub.Client
	storageBucketName string
	storageBucket     *storage.BucketHandle
	datastoreClient   *datastore.Client
}

func (h *uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		ctx, cancelCtx := context.WithCancel(r.Context())

		defer cancelCtx()

		uploadName := r.URL.Query().Get("name")

		sessionToken := r.URL.Query().Get("token")

		uploadType := r.Header.Get(http.CanonicalHeaderKey("Content-Type"))

		var bodyReader io.Reader = r.Body

		user, email, err := getUser(ctx, h.datastoreClient, sessionToken)

		if err != nil {
			log.Println("Invalid Token")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		topic := createTopic(ctx, h.pubsubClient, user)

		bodyBuffer := new(bytes.Buffer)

		if uploadType == clipboardMimeType {
			bodyReader = io.TeeReader(bodyReader, bodyBuffer)
		}

		objectName := fmt.Sprintf("%s/%016x", user, time.Now().UnixNano())

		obj := h.storageBucket.Object(objectName)

		objWriter := obj.NewWriter(ctx)

		if _, err := io.Copy(objWriter, bodyReader); err != nil {
			log.Println("Error streaming data:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := objWriter.Close(); err != nil {
			log.Println("Error closing writer:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		objectAttrsToUpdate := storage.ObjectAttrsToUpdate{
			ACL: []storage.ACLRule{
				{
					Entity: storage.ACLEntity(fmt.Sprint("user-", email)),
					Role:   storage.RoleReader,
				},
			},
			Metadata: map[string]string{
				metaDataName: uploadName,
			},
		}

		if uploadType != "" {
			objectAttrsToUpdate.ContentType = uploadType
		}

		if _, err := obj.Update(context.Background(), objectAttrsToUpdate); err != nil {
			log.Println("Error updating file attributes:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		newAttrs, err := obj.Attrs(context.Background())

		if err != nil {
			log.Println("Error getting object attributes")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		notificationData, err := json.Marshal(createFileNotification(h.storageBucketName, newAttrs, bodyBuffer))

		if err != nil {
			log.Println("Error marshalling notification:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		topic.Publish(context.Background(), &pubsub.Message{Data: notificationData})

		fmt.Fprintln(w, "Done")

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//UploadHandler handles the uploading of new files
func UploadHandler(googleLoginAppID string, pubsubClient *pubsub.Client, storageBucketName string, storageBucket *storage.BucketHandle, datastoreClient *datastore.Client) (string, http.Handler) {
	return uploadPath, &uploadHandler{
		googleLoginAppID:  googleLoginAppID,
		pubsubClient:      pubsubClient,
		storageBucketName: storageBucketName,
		storageBucket:     storageBucket,
		datastoreClient:   datastoreClient,
	}
}
