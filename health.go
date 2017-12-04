package main

import (
	"fmt"
	"net/http"
)

const healthPath = "/_ah/health"

type healthHandler struct {
}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Sprint(w, "OK")
}

//HealthHandler handles health checks
func HealthHandler() (string, http.Handler) {
	return healthPath, &healthHandler{}
}
