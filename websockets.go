package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"

	"github.com/gorilla/websocket"
)

const websocketPath = "/ws"

type websocketHandler struct {
	pubsubClient  *pubsub.Client
	storageBucket *storage.BucketHandle
}

type uploadHeader struct {
	Name string
	Type string
}

type fileNotification struct {
	Name    string
	Type    string
	Created int64
	URL     string
	Body    string
}

func (h *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx, cancelCtx := context.WithCancel(r.Context())

		defer cancelCtx()

		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade to ws:", err)
			return
		}

		keys, err := GoogleKeys()

		if err != nil {
			log.Println("Failed to fetch google public keys:", err)
			return
		}

		messageType, payload, err := conn.ReadMessage()

		if err != nil {
			log.Println("Error reading token:", err)
			return
		}

		if messageType != websocket.TextMessage {
			log.Println("Token must be text")
			return
		}

		userID, userEmail, err := VerifyToken(string(payload), keys, "812818444262-dihtcq1cl07rrc4d3gs86obfs95dhe4i.apps.googleusercontent.com")

		if err != nil {
			log.Println("Auth Error:", err)
			return
		}

		log.Println("User", userID, "logged in")

		userIDHash := sha256.Sum256([]byte(fmt.Sprint(userID)))

		topicName := fmt.Sprintf("notifications-%x", userIDHash)

		topic, err := h.pubsubClient.CreateTopic(ctx, topicName)

		if err != nil {
			topic = h.pubsubClient.Topic(topicName)
		}

		go func() {
			defer cancelCtx()
			for {
				var header uploadHeader

				if err := conn.ReadJSON(&header); err != nil {
					log.Println("Error getting header:", err)
					return
				}

				_, dataReader, err := conn.NextReader()

				bodyBuffer := new(bytes.Buffer)

				if header.Type == "text/x-clipboard" {
					dataReader = io.TeeReader(dataReader, bodyBuffer)
				}

				if err != nil {
					log.Println("Error getting next reader:", err)
					return
				}

				objectName := fmt.Sprintf("%x/%x", userIDHash, time.Now().UnixNano())

				obj := h.storageBucket.Object(objectName)

				objWriter := obj.NewWriter(ctx)

				if _, err := io.Copy(objWriter, dataReader); err != nil {
					log.Println("Error streaming data:", err)
					return
				}

				if err := objWriter.Close(); err != nil {
					log.Println("Error closing writer:", err)
					return
				}

				if _, err := obj.Update(context.Background(), storage.ObjectAttrsToUpdate{
					ACL: []storage.ACLRule{
						{
							Entity: storage.ACLEntity(fmt.Sprint("user-", userEmail)),
							Role:   storage.RoleReader,
						},
					},
					Metadata: map[string]string{
						"x-name": header.Name,
					},
					ContentType: header.Type,
				}); err != nil {
					log.Println("Error updating file attributes:", err)
					return
				}

				newAttrs, err := obj.Attrs(context.Background())

				if err != nil {
					log.Println("Error getting object attributes")
				}

				notification := &fileNotification{
					Name:    header.Name,
					Type:    newAttrs.ContentType,
					Created: newAttrs.Created.UTC().UnixNano() / 1000000,
					URL:     fmt.Sprint("https://storage.cloud.google.com/cloud-computing-coursework-storage/", objectName),
					Body:    string(bodyBuffer.Bytes()),
				}

				notificationData, err := json.Marshal(notification)

				if err != nil {
					log.Println("Error marshalling notification:", err)
					return
				}

				topic.Publish(context.Background(), &pubsub.Message{Data: notificationData})
			}
		}()

		subName := fmt.Sprintf("listen-%x-%x", userIDHash, time.Now().UnixNano())

		sub, err := h.pubsubClient.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{Topic: topic})

		if err != nil {
			log.Println("Failed to create subscriber:", err)
			return
		}

		defer func() {
			if err := sub.Delete(context.Background()); err != nil {
				log.Println("Could not delete sub:", err)
			}
		}()

		eventLock := &sync.Mutex{}

		err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			eventLock.Lock()
			defer eventLock.Unlock()
			defer m.Ack()

			if err := conn.WriteMessage(websocket.TextMessage, m.Data); err != nil {
				log.Println("Error writing message:", err)
			}
		})

		if err != nil {
			log.Println("Error receiving messages")
		}

		log.Println("User", userID, "disconnected")

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

//WebsocketHandler handles the transmission of a new snippet
func WebsocketHandler(pubsubClient *pubsub.Client, storageBucket *storage.BucketHandle) (string, http.Handler) {
	return websocketPath, &websocketHandler{
		pubsubClient:  pubsubClient,
		storageBucket: storageBucket,
	}
}
