package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"bufio"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const uploadPath = "/upload"

type uploadHeader struct {
	Name  string
	Type  string
	Token string
}

type uploadHandler struct {
	googleLoginAppID  string
	pubsubClient      *pubsub.Client
	storageBucketName string
	storageBucket     *storage.BucketHandle
	datastoreClient   *datastore.Client
}

func readHeader(r *http.Request) (header *uploadHeader, bodyReader io.Reader, err error) {
	bufBodyReader := bufio.NewReader(r.Body)

	bodyReader = bufBodyReader

	headerString, err := bufBodyReader.ReadString('\n')

	if err != nil {
		return
	}

	headerBytes, err := base64.StdEncoding.DecodeString(headerString)

	if err != nil {
		return
	}

	header = new(uploadHeader)

	err = json.Unmarshal(headerBytes, header)

	return
}

func (h *uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		ctx, cancelCtx := context.WithCancel(r.Context())

		defer cancelCtx()

		header, bodyReader, err := readHeader(r)

		if err != nil {
			log.Println("Error reading header:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		user, email, err := getUser(ctx, h.datastoreClient, header.Token)

		if err != nil {
			log.Println("Invalid Token")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		topic, err := createTopic(ctx, h.pubsubClient, user)

		if err != nil {
			log.Println("Error creating topic:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		bodyBuffer := new(bytes.Buffer)

		if header.Type == clipboardMimeType {
			bodyReader = io.TeeReader(bodyReader, bodyBuffer)
		}

		objectName := fmt.Sprintf("%x/%016x", user, time.Now().UnixNano())

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
				metaDataName: header.Name,
			},
		}

		if header.Type != "" {
			objectAttrsToUpdate.ContentType = header.Type
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
