package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ishaan-Buddharaju/open-gallery/injest"
	"github.com/Ishaan-Buddharaju/open-gallery/types"
	"github.com/joho/godotenv"
	"google.golang.org/api/gmail/v1"
)

func main() {
	ctx := context.Background()

	var (
		gmailClient *gmail.Service
		err         error
	)
	gmailClient, err = injest.SetupOauthClient(ctx, "credentials.json", "token.json")
	if err != nil {
		fmt.Printf("Error on OauthSetup: %v", err)
	}
	result, err := injest.FetchInbox(gmailClient, "")
	if err != nil {
		fmt.Printf("Error on inbox fetch: %v", err)
	}
	fmt.Printf("Raw Response Struct:\n%+v\n", result)

	// for _, msgSummary := range result.Messages {
	// 	// Fetch the full details for this specific message ID
	// 	msg, err := gmailService.Users.Messages.Get("me", msgSummary.Id).Format("full").Do()
	// 	if err != nil {
	// 		fmt.Printf("Unable to retrieve message %s: %v", msgSummary.Id, err)
	// 		continue
	// 	}
	// 	fmt.Println("----------------------------------------")
	// 	fmt.Printf("Message ID: %s\n", msg.Id)
	// 	fmt.Printf("Snippet:    %s\n", msg.Snippet) // Brief preview of text

	// 	// Extract the Subject from the headers block
	// 	for _, header := range msg.Payload.Headers {
	// 		if header.Name == "Subject" {
	// 			fmt.Printf("Subject:    %s\n", header.Value)
	// 		}
	// 		if header.Name == "From" {
	// 			fmt.Printf("From:       %s\n", header.Value)
	// 		}
	// 	}
	// }

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

	store, err := types.NewCursorStore(os.Getenv("CursorStorePath"))
	if err != nil {
		log.Fatalf("Failed to create cursor store: %v", err)
	}
	err = injest.ReceiveGmailNotifications(ctx, subClient, gmailClient, store)
	if err != nil {
		log.Fatalf("Failed to receive Gmail notifications: %v", err)
	} else {
		log.Printf("Gmail notifications received successfully")
	}
}
