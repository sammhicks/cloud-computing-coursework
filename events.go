package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	oldContext "golang.org/x/net/context"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const eventsPath = "/events"

type eventsHandler struct {
	ctx               context.Context
	googleLoginAppID  string
	pubsubClient      *pubsub.Client
	storageBucketName string
	storageBucket     *storage.BucketHandle
	datastoreClient   *datastore.Client
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx, closeFunc := context.WithCancel(r.Context())

		defer closeFunc()

		go func() {
			select {
			case <-h.ctx.Done():
				closeFunc()
			case <-ctx.Done():
			}
		}()

		f, ok := w.(http.Flusher)

		if !ok {
			log.Println("Cannot create flusher")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		identityToken := r.URL.Query().Get("token")

		keys, err := GoogleKeys()

		if err != nil {
			log.Println("Failed to fetch google public keys:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		userID, userEmail, err := VerifyToken(identityToken, keys, h.googleLoginAppID)

		if err != nil {
			log.Println("Auth Error:", err)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		userIDHashBytes := sha256.Sum256([]byte(fmt.Sprint(userID)))

		userIDHash := base64.RawURLEncoding.EncodeToString(userIDHashBytes[:])

		subscription, err := createSubscription(ctx, h.pubsubClient, userIDHash)

		if err != nil {
			log.Println("Failed to generate subscription:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		sessionToken, err := genToken(ctx, h.datastoreClient, userIDHash, userEmail)

		if err != nil {
			log.Println("Failed to generate token:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "data: %s\n\n", sessionToken)

		f.Flush()

		log.Println("User", userID, "logged in")

		defer func() { log.Println("User", userID, "disconnected") }()

		objIter := h.storageBucket.Objects(ctx, &storage.Query{Prefix: userIDHash})

		for {
			objAttrs, err := objIter.Next()

			if err == iterator.Done {
				break
			}

			if err != nil {
				log.Println("Error fetching object from history:", err)
				return
			}

			bodyBuffer := new(bytes.Buffer)

			if objAttrs.ContentType == clipboardMimeType {
				obj := h.storageBucket.Object(objAttrs.Name)

				objReader, err := obj.NewReader(ctx)

				if err != nil {
					log.Println("Error getting object reader:", err)
					return
				}

				if _, err := io.Copy(bodyBuffer, objReader); err != nil {
					log.Println("Error reading object data from history:", err)
					return
				}
			}

			notificationData, err := json.Marshal(createFileNotification(h.storageBucketName, objAttrs, bodyBuffer))

			if err != nil {
				log.Println("Error marshalling notification:", err)
			}

			fmt.Fprintf(w, "data: %s\n\n", base64.StdEncoding.EncodeToString(notificationData))

			f.Flush()
		}

		eventStreamLock := &sync.Mutex{}

		if err := subscription.Receive(ctx, func(ctx oldContext.Context, m *pubsub.Message) {
			eventStreamLock.Lock()
			defer eventStreamLock.Unlock()
			defer m.Ack()

			fmt.Fprintf(w, "data: %s\n\n", base64.StdEncoding.EncodeToString(m.Data))
			f.Flush()
		}); err != nil {
			log.Println("Error receiving messages:", err)
		}

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//EventsHandler handles notifying clients of events
func EventsHandler(ctx context.Context, googleLoginAppID string, pubsubClient *pubsub.Client, storageBucketName string, storageBucket *storage.BucketHandle, datastoreClient *datastore.Client) (string, http.Handler) {
	return eventsPath, &eventsHandler{
		ctx:               ctx,
		googleLoginAppID:  googleLoginAppID,
		pubsubClient:      pubsubClient,
		storageBucketName: storageBucketName,
		storageBucket:     storageBucket,
		datastoreClient:   datastoreClient,
	}
}
