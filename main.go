package main

import (
	"fmt"
	"google.golang.org/api/gmail/v1"
	"context"
	"log"
	"os"
	"github.com/Ishaan-Buddharaju/open-gallery/injest"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	var (
		gmailService *gmail.Service
		err error
	)
	gmailService, err = injest.SetupOauthClient(ctx, "credentials.json", "token.json")
	if err != nil {
		fmt.Printf("Error on OauthSetup: %v", err)
	}
	result, err := injest.FetchInbox(gmailService, "")
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
	sub, err := injest.SetupPubSubClient(ctx, projectID, topicName, subName)
	if err != nil {
		log.Fatalf("Failed client initialization: %v", err)
	}
	log.Println("Pub/Sub Subscriber worked: %s", sub.ID())
}
