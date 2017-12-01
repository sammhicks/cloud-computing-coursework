package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
)

func createTopic(ctx context.Context, client *pubsub.Client, userIDHash string) (*pubsub.Topic, error) {
	return client.CreateTopic(ctx, fmt.Sprintf("notifications-", userIDHash))
}

func createSubscription(ctx context.Context, client *pubsub.Client, userIDHash string) (sub *pubsub.Subscription, err error) {
	topic, err := createTopic(ctx, client, userIDHash)

	if err != nil {
		return
	}

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

		log.Println("Deleted subscription")
	}()

	return
}
