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

	oldContext "golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/gorilla/websocket"
)

const websocketPath = "/ws"

type uploadHeader struct {
	Name  string
	Type  string
	Token string
}

type websocketHandler struct {
	googleLoginAppID  string
	pubsubClient      *pubsub.Client
	storageBucketName string
	storageBucket     *storage.BucketHandle
}

func (h *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx, cancelCtx := context.WithCancel(r.Context())

		defer cancelCtx()

		w.Header().Set("X-Accel-Buffering", "no")

		hBuf := new(bytes.Buffer)

		if r.Header.Write(hBuf) != nil {
			log.Println("Header Error")
		} else {
			log.Println(string(hBuf.Bytes()))
		}

		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade to ws:", err)
			return
		}

		defer conn.Close()

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

		userID, userEmail, err := VerifyToken(string(payload), keys, h.googleLoginAppID)

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

				if err != nil {
					log.Println("Error getting next reader:", err)
					return
				}

				bodyBuffer := new(bytes.Buffer)

				if header.Type == clipboardMimeType {
					dataReader = io.TeeReader(dataReader, bodyBuffer)
				}

				objectName := fmt.Sprintf("%x/%016x", userIDHash, time.Now().UnixNano())

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
						metaDataName: header.Name,
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

				notificationData, err := json.Marshal(createFileNotification(h.storageBucketName, newAttrs, bodyBuffer))

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

		websocketLock := &sync.Mutex{}

		go func() {
			objIter := h.storageBucket.Objects(ctx, &storage.Query{
				Prefix: fmt.Sprintf("%x", userIDHash),
			})

			for {
				objAttrs, err := objIter.Next()

				if err == iterator.Done {
					break
				}

				if err != nil {
					log.Println("Error fetching object from history:", err)
					cancelCtx()
					return
				}

				bodyBuffer := new(bytes.Buffer)

				if objAttrs.ContentType == clipboardMimeType {
					obj := h.storageBucket.Object(objAttrs.Name)

					objReader, err := obj.NewReader(ctx)

					if err != nil {
						log.Println("Error getting object reader:", err)
						cancelCtx()
						return
					}

					if _, err := io.Copy(bodyBuffer, objReader); err != nil {
						log.Println("Error reading object data from history:", err)
						cancelCtx()
						return
					}
				}

				notification := createFileNotification(h.storageBucketName, objAttrs, bodyBuffer)

				websocketLock.Lock()
				err = conn.WriteJSON(notification)
				websocketLock.Unlock()

				if err != nil {
					log.Println("Error sending notification:", err)
					cancelCtx()
					return
				}
			}
		}()

		err = sub.Receive(ctx, func(ctx oldContext.Context, m *pubsub.Message) {
			websocketLock.Lock()
			defer websocketLock.Unlock()
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
func WebsocketHandler(googleLoginAppID string, pubsubClient *pubsub.Client, storageBucketName string, storageBucket *storage.BucketHandle) (string, http.Handler) {
	return websocketPath, &websocketHandler{
		googleLoginAppID:  googleLoginAppID,
		pubsubClient:      pubsubClient,
		storageBucketName: storageBucketName,
		storageBucket:     storageBucket,
	}
}
