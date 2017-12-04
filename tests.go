package main

import (
	"fmt"
	"net/http"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
)

const testPath = "/test"

type testHandler struct {
	googleLoginAppID  string
	pubsubClient      *pubsub.Client
	storageBucketName string
	storageBucket     *storage.BucketHandle
	datastoreClient   *datastore.Client
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		/*ctx, closeFunc := context.WithCancel(r.Context())

		defer closeFunc()

		datastoreClient, err := datastore.NewClient(ctx, "cloud-computing-coursework")

		if err != nil {
			log.Println("Client:", err)
			return
		}

		key := datastore.IncompleteKey("SessionCookie", nil)

		_, err = datastoreClient.Put(ctx, key, &sessionCookie{
			User:   "Me",
			Cookie: "MyCookie",
		})

		if err != nil {
			log.Println("Put:", err)
			return
		}*/

		fmt.Sprint("Done")
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//TestHandler tests stuff
func TestHandler(googleLoginAppID string, pubsubClient *pubsub.Client, storageBucketName string, storageBucket *storage.BucketHandle) (string, http.Handler) {
	return testPath, &testHandler{
		googleLoginAppID:  googleLoginAppID,
		pubsubClient:      pubsubClient,
		storageBucketName: storageBucketName,
		storageBucket:     storageBucket,
	}
}
