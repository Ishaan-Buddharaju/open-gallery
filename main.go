package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"net/http"

	"github.com/Ishaan-Buddharaju/open-gallery/config"
	"github.com/Ishaan-Buddharaju/open-gallery/ingest"
	"github.com/Ishaan-Buddharaju/open-gallery/storage"
	"github.com/Ishaan-Buddharaju/open-gallery/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	gmailClient, err := ingest.SetupOauthClient(ctx, cfg.CredsFile, cfg.TokenFile)
	if err != nil {
		return fmt.Errorf("oauth setup: %w", err)
	}

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	var mode string
	db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	log.Printf("db=%s journal_mode=%s", cfg.DBPath, mode)

	subClient, err := ingest.SetupPubSubClient(ctx, cfg.ProjectID, cfg.TopicName, cfg.SubName)
	if err != nil {
		return fmt.Errorf("pubsub client: %w", err)
	}
	log.Printf("Pub/Sub Subscriber worked: %s", subClient.ID())

	watchResp, err := ingest.WatchTopic(ctx, gmailClient, cfg.ProjectID, cfg.TopicName)
	if err != nil {
		return fmt.Errorf("gmail watch: %w", err)
	}
	if err := storage.SeedCursor(db, watchResp.HistoryId, "gmail"); err != nil {
		return fmt.Errorf("seed cursor: %w", err)
	}
	log.Printf("Watch response: %+v", watchResp)

	srv := web.New(db, cfg.HTTPAddr, cfg.ImageDir)

	var wg sync.WaitGroup
	wg.Go(func() {
		ingest.RenewWatch(ctx, gmailClient, cfg.ProjectID, cfg.TopicName)
	})
	wg.Go(func() {
		if err := ingest.ReceiveGmailNotifications(ctx, subClient, gmailClient, db, cfg.ImageDir); err != nil {
			log.Printf("gmail ingest stopped: %v", err)
		}
	})
	wg.Go(func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("web server stopped: %v", err)
		}
	})

	<-ctx.Done()
	log.Printf("shutting down")
	srv.Shutdown(context.Background())
	wg.Wait()
	return nil
}
