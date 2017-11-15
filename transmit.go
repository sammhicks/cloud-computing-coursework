package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
)

const transmitPath = "/transmit"

type transmitHandler struct {
}

func (h *transmitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		ctx := r.Context()
		client, err := storage.NewClient(ctx)
		if err != nil {
			log.Println("Error creating storage client:", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		defer client.Close()

		bkt := client.Bucket("cloud-computing-coursework.appspot.com")

		objWriter := bkt.Object(fmt.Sprintf("test/%d", time.Now().Unix())).NewWriter(ctx)

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

		fmt.Fprintln(w, "Successfuly transmitted snippet")
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//TransmitHandler handles the transmission of a new snippet
func TransmitHandler() (string, http.Handler) { return transmitPath, &transmitHandler{} }
