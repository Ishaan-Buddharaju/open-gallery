package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ishaan-Buddharaju/open-gallery/injest"
	"github.com/joho/godotenv"
	"google.golang.org/api/gmail/v1"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		gmailClient *gmail.Service
		err         error
	)
	gmailClient, err = injest.SetupOauthClient(ctx, "credentials.json", "token.json")
	if err != nil {
		fmt.Printf("Error on OauthSetup: %v", err)
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env")
	}

	projectID := os.Getenv("GCloudProjectID")
	topicName := os.Getenv("GCloudTopicName")
	subName := os.Getenv("GCloudGmailSubscription")
	subClient, err := injest.SetupPubSubClient(ctx, projectID, topicName, subName)
	if err != nil {
		log.Fatalf("Failed client initialization: %v", err)
	}
	log.Printf("Pub/Sub Subscriber worked: %s", subClient.ID())

	watchResp, err := injest.WatchTopic(ctx, gmailClient, projectID, topicName)
	if err != nil {
		log.Fatalf("Failed to subscribe to Gmail: %v", err)
	}
	log.Printf("Watch response: %+v\n", watchResp)
	go func() { // renew every day (7 days to expiration)
		t := time.NewTicker(24 * time.Hour)
		for range t.C {
			_, err := injest.WatchTopic(ctx, gmailClient, projectID, topicName)
			if err != nil {
				log.Printf("Failed to renew subscription: %v", err)
			}
		}
	}()

	if err != nil {
		log.Fatalf("Failed to create cursor store: %v", err)
	}
	err = injest.ReceiveGmailNotifications(ctx, subClient, gmailClient)
	if err != nil {
		log.Fatalf("Failed to receive Gmail notifications: %v", err)
	} else {
		log.Printf("Gmail notifications received successfully")
	}
}
