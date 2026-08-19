package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Ishaan-Buddharaju/open-gallery/ingest"
	"github.com/Ishaan-Buddharaju/open-gallery/storage"
	"github.com/joho/godotenv"
	"google.golang.org/api/gmail/v1"
)

//TODODODODODO Add cursor syncing

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		gmailClient *gmail.Service
		err         error
	)
	gmailClient, err = ingest.SetupOauthClient(ctx, "credentials.json", "token.json")
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
	subClient, err := ingest.SetupPubSubClient(ctx, projectID, topicName, subName)
	if err != nil {
		log.Fatalf("Failed client initialization: %v", err)
	}
	log.Printf("Pub/Sub Subscriber worked: %s", subClient.ID())

	//Setup database
	db, err := storage.Open(os.Getenv("DB_PATH"))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	var mode string
	db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	log.Printf("db=%s journal_mode=%s", "opengallery", mode)
	defer db.Close()

	watchResp, err := ingest.WatchTopic(ctx, gmailClient, projectID, topicName)
	if err != nil {
		log.Fatalf("Failed to subscribe to Gmail: %v", err)
	}
	err = storage.SeedCursor(db, watchResp.HistoryId, "gmail")
	if err != nil {
		log.Fatalf("SeedCursor failed: %v", err)
	}
	log.Printf("Watch response: %+v\n", watchResp)
	go func() { // renew every day (7 days to expiration)
		t := time.NewTicker(24 * time.Hour)
		for range t.C {
			_, err := ingest.WatchTopic(ctx, gmailClient, projectID, topicName)
			if err != nil {
				log.Printf("Failed to renew subscription: %v", err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := ingest.ReceiveGmailNotifications(ctx, subClient, gmailClient, db); err != nil {
			log.Printf("gmail ingest stopped: %v", err)
		}
	})

	<-ctx.Done()
	log.Printf("shutting down")
	wg.Wait()
}
