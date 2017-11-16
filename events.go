package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const eventsPath = "/events"

type eventsHandler struct {
	pubsubClient  *pubsub.Client
	storageBucket *storage.BucketHandle
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx := r.Context()

		keys, err := GoogleKeys()

		if err != nil {
			log.Println("Failed to fetch google public keys:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		userID, _, err := VerifyToken(r.URL.Query().Get("token"), keys, "812818444262-dihtcq1cl07rrc4d3gs86obfs95dhe4i.apps.googleusercontent.com")

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

		subName := fmt.Sprintf("listen-%x-%x", userID, time.Now().UnixNano())

		sub, err := h.pubsubClient.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{Topic: topic})

		if err != nil {
			log.Println("Failed to create subscriber:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		defer func() {
			if err := sub.Delete(context.Background()); err != nil {
				log.Println("Could not delete sub:", err)
			}
		}()

		f, ok := w.(http.Flusher)
		if !ok {
			log.Println("Failed to create flusher:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		w.Write([]byte(""))

		f.Flush()

		eventLock := &sync.Mutex{}

		err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			eventLock.Lock()
			defer eventLock.Unlock()
			defer m.Ack()

			fmt.Fprint(w, "data:", string(m.Data), "\n\n")

			f.Flush()
		})

		if err != nil {
			log.Println("Failed to listen for messages:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		log.Println("Event stream closed from client")

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//EventsHandler handles the streaming of new snippets arriving
func EventsHandler(pubsubClient *pubsub.Client, storageBucket *storage.BucketHandle) (string, http.Handler) {
	return eventsPath, &eventsHandler{
		pubsubClient:  pubsubClient,
		storageBucket: storageBucket,
	}
}
