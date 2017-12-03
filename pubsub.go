package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
)

func createTopic(ctx context.Context, client *pubsub.Client, userIDHash string) *pubsub.Topic {
	topicName := fmt.Sprint("notifications-", userIDHash)

	topic, err := client.CreateTopic(ctx, topicName)

	if err != nil {
		topic = client.Topic(topicName)
	}

	return topic
}

func createSubscription(ctx context.Context, client *pubsub.Client, userIDHash string) (sub *pubsub.Subscription, err error) {
	topic := createTopic(ctx, client, userIDHash)

	subName := fmt.Sprintf("listen-%s-%016x", userIDHash, time.Now().UnixNano())

	sub, err = client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{Topic: topic})

	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		if err := sub.Delete(context.Background()); err != nil {
			log.Println("Failed to delete subscription:", err)
			return
		}
	}()

	return
}
