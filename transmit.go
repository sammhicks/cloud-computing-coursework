package main

import (
	"fmt"
	"net/http"
)

const transmitPath = "/transmit"

type transmitHandler struct {
}

func (h *transmitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		fmt.Fprintln(w, "Successfuly transmitted snippet")
	default:
		status := http.StatusMethodNotAllowed
		w.WriteHeader(status)
		fmt.Fprintln(w, http.StatusText(status))
	}
}

//TransmitHandler handles the transmission of a new snippet
func TransmitHandler() (string, http.Handler) { return transmitPath, &transmitHandler{} }
