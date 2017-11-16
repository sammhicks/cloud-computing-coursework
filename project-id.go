package main

import (
	"errors"
	"os"
)

// ProjectID fetches the project id from environment variables
func ProjectID() (projectID string, err error) {
	projectID, projectIDDeclared := os.LookupEnv("project")

	if !projectIDDeclared {
		err = errors.New("Project ID not declared")
	}

	return
}
