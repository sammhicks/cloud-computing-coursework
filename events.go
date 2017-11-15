package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
)

const eventsPath = "/events"

type eventsHandler struct {
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx := r.Context()

		projectID, projectIDDeclared := os.LookupEnv("project")

		if !projectIDDeclared {
			log.Println("Project ID not declared")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		client, err := pubsub.NewClient(ctx, projectID)
		if err != nil {
			log.Println("Failed to create client:", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		topic := client.TopicInProject("test", projectID)

		sub, err := client.CreateSubscription(ctx, "test-sub", pubsub.SubscriptionConfig{Topic: topic})

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

		f.Flush()

		eventLock := &sync.Mutex{}

		err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			eventLock.Lock()
			defer eventLock.Unlock()
			defer m.Ack()

			fmt.Fprint(w, "data:")

			e := base64.NewEncoder(base64.StdEncoding, w)

			bw := bufio.NewWriter(e)

			if _, err := bw.Write(m.Data); err != nil {
				log.Println("Error writing message body", err)
				return
			}

			if err := bw.Flush(); err != nil {
				log.Println("Error flushing message body", err)
				return
			}

			if err := e.Close(); err != nil {
				log.Println("Error closing base64 writer", err)
				return
			}

			if _, err := fmt.Fprint(w, "\n\n"); err != nil {
				log.Println("Error finialising event", err)
				return
			}

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
func EventsHandler() (string, http.Handler) { return eventsPath, &eventsHandler{} }
