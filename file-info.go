package main

import (
	"bytes"
	"fmt"

	"cloud.google.com/go/storage"
)

const storageURL = "https://storage.cloud.google.com"

const clipboardMimeType = "text/x-clipboard"

const metaDataName = "x-name"

type fileNotification struct {
	Name    string
	Type    string
	Created int64
	URL     string
	Body    string
}

func createFileNotification(bucketName string, objAttrs *storage.ObjectAttrs, body *bytes.Buffer) *fileNotification {
	return &fileNotification{
		Name:    objAttrs.Metadata[metaDataName],
		Type:    objAttrs.ContentType,
		Created: objAttrs.Created.UTC().UnixNano() / 1000000,
		URL:     fmt.Sprintf("%s/%s/%s", storageURL, bucketName, objAttrs.Name),
		Body:    string(body.Bytes()),
	}
}
