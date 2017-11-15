package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

const eventsPath = "/events"

type eventsHandler struct {
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		ctx := r.Context()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		f.Flush()

		messages := make(chan Snippet)

		go func() {
			for i := 0; i < 20; i++ {
				messages <- Snippet{
					Body: []byte("Message: " + strconv.FormatInt(int64(i), 10))}

				time.Sleep(200 * time.Millisecond)
			}

			close(messages)
		}()

		for {
			select {
			case <-ctx.Done():
				log.Println("Event stream closed from client")
				return
			case m, ok := <-messages:
				if !ok {
					log.Println("No More messages")
					return
				}
				fmt.Fprint(w, "data:")

				e := base64.NewEncoder(base64.StdEncoding, w)

				bw := bufio.NewWriter(e)

				if _, err := bw.Write(m.Body); err != nil {
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
			}
		}

	default:
		status := http.StatusMethodNotAllowed
		w.WriteHeader(status)
		fmt.Fprintln(w, http.StatusText(status))
	}
}

//EventsHandler handles the streaming of new snippets arriving
func EventsHandler() (string, http.Handler) { return eventsPath, &eventsHandler{} }
