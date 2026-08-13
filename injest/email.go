package injest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/Ishaan-Buddharaju/open-gallery/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* Gmail Auth & Credential setup funcs */

func SetupOauthClient(ctx context.Context, credsFile string, tokenFile string) (*gmail.Service, error) {
	tokenSource, err := setupGmailTokenSource(ctx, credsFile, tokenFile)
	if err != nil {
		log.Printf("Setup failed: %v", err)
		return nil, err
	}

	gmailService, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Printf("Failed to initialize Gmail client: %v", err)
		return nil, err
	}

	return gmailService, nil
}

func setupGmailTokenSource(ctx context.Context, credsFile string, tokenFile string) (oauth2.TokenSource, error) {
	b, err := os.ReadFile(credsFile)
	if err != nil { // Corrected syntax: explicitly checking for != nil
		return nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse configuration file: %w", err)
	}

	token, err := tokenFromFile(tokenFile)
	if err != nil {
		// If the file is missing or unreadable, trigger the browser workflow
		token = getTokenFromWeb(config)
		saveToken(tokenFile, token)
	}

	return config.TokenSource(ctx, token), nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// 1. Create a channel to pass the token back from the HTTP handler
	codeChan := make(chan string)

	// 2. Define the local callback server handler
	server := &http.Server{Addr: ":8080"} // Must match the port in your Google Cloud Console redirect URI

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Get the code parameter from the redirect URL
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprint(w, "Authentication failed: missing authorization code.")
			return
		}

		// Send user a success message in browser
		fmt.Fprint(w, "Authentication successful! You can close this browser tab and return to your terminal.")

		// Pass code back to the main routine
		codeChan <- code
	})

	// 3. Start the local server in a separate background goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Local server error: %v", err)
		}
	}()

	// 4. Print the URL for the user to follow
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser to authorize the app: \n%v\n", authURL)

	// 5. Block execution here until the browser submits the code to our channel
	authCode := <-codeChan

	// 6. Shut down the temporary web server cleanly
	_ = server.Shutdown(context.Background())

	// 7. Exchange the short-lived authorization code for the actual tokens
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()

	json.NewEncoder(f).Encode(token)
}

/* PubSub Setup Helpers */

func SetupPubSubClient(ctx context.Context, projectID string, topicName string, subName string) (*pubsub.Subscriber, error) {
	// TODO check restart logic in main to gracefully handle shutdown and webhook
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Pub/Sub Client: %w", err)
	}
	topicPath := fmt.Sprintf("projects/%s/topics/%s", projectID, topicName)
	pbTopic := &pubsubpb.Topic{
		Name: topicPath,
	}
	_, err = client.TopicAdminClient.CreateTopic(ctx, pbTopic)

	if err != nil && !IsAlreadyExists(err) {
		return nil, fmt.Errorf("topic creation failed: %w", err)
	}

	subPath := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subName)
	pbSubscription := &pubsubpb.Subscription{
		Name:                      subPath,
		Topic:                     topicPath,
		AckDeadlineSeconds:        30,
		RetainAckedMessages:       false,
		EnableExactlyOnceDelivery: false,
	}
	_, err = client.SubscriptionAdminClient.CreateSubscription(ctx, pbSubscription)
	if err != nil && !IsAlreadyExists(err) {
		return nil, fmt.Errorf("subscription creation failed: %w", err)
	}

	sub := client.Subscriber(subPath)
	if err != nil && !IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to initialize subscriber runtime: %w", err)
	}

	sub.ReceiveSettings.NumGoroutines = 4
	sub.ReceiveSettings.MaxOutstandingMessages = 20

	return sub, nil
}

func WatchTopic(ctx context.Context, gmailClient *gmail.Service, projectID string, topicName string) (*gmail.WatchResponse, error) {
	topicName = "projects/" + projectID + "/topics/" + topicName
	watchReq := gmailClient.Users.Watch("me", &gmail.WatchRequest{
		LabelIds:            []string{"INBOX"},
		LabelFilterBehavior: "INCLUDE",
		TopicName:           topicName,
	})

	watchResp, err := watchReq.Do()
	if err != nil {
		log.Printf("Failed to subscribe to Gmail: %v", err)
		return nil, err
	}
	return watchResp, nil
}

func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.AlreadyExists
}

/* Notification Processing from PubSub */

func ReceiveGmailNotifications(ctx context.Context, subClient *pubsub.Subscriber, srv *gmail.Service) error {
	handler := func(ctx context.Context, msg *pubsub.Message) {
		var n types.GmailNotification
		if err := json.Unmarshal(msg.Data, &n); err != nil {
			log.Printf("unparseable payload %q: %v", msg.Data, err)

			msg.Ack()
			return
		}
		n.ReceivedAt = msg.PublishTime
		log.Printf("Received notification (historyId=%d)", n.HistoryId)
		if err := process(ctx, srv, n); err != nil {
			log.Printf("process failed (historyId=%d): %v", n.HistoryId, err)
			msg.Nack()
			return
		}
		msg.Ack()
	}
	log.Printf("Recieving Gmail notifications now")
	return subClient.Receive(ctx, handler)
}

func process(ctx context.Context, srv *gmail.Service, n types.GmailNotification) error {
	newMessages, err := listNewMessages(ctx, srv, n.HistoryId)
	log.Printf("Found %d new messages in processing", len(newMessages))
	if err != nil {
		return err
	}
	for _, msg := range newMessages {
		fmt.Println(msg)
	}
	return nil
}

/* Notification Processing after PubSub notification */

// func ProcessNotification(ctx context.Context, srv *gmail.Service, history string) error {
// TODO
// }

// func handleNewMessage(ctx context.Context, srv *gmail.Service, id string) error {
// 	msg, err := srv.Users.Messages.Get("me", id).Format("full").Context(ctx).Do()
// 	if err != nil {
// 		return err
// 	}
//
// 	log.Printf("%s: %s", id, msg.Snippet)
// 	return nil
// }

// // TODO Add Oauth Device flow for the projector to setup the source inbox plus filters
func listNewMessages(ctx context.Context, gmailService *gmail.Service, historyID uint64) ([]*gmail.Message, error) {
	var req *gmail.UsersHistoryListCall = gmailService.Users.History.List("me").
		StartHistoryId(historyID).
		HistoryTypes("messageAdded")
	result, err := req.Do()
	log.Printf("History result: %v", result)
	if err != nil {
		return nil, err
	}
	var newMessages []*gmail.Message
	for _, h := range result.History {
		for _, m := range h.Messages {
			newMessages = append(newMessages, m)
			log.Printf("Found message %s", m.Id)
		}
	}
	return newMessages, nil

}

// func processNewMessages(ctx context.Context, srv *gmail.Service, newMessages []*gmail.Message) (error) {

// }

// func processMessage(ctx context.Context, srv *gmail.Service, msg *gmail.Message) error {
// 	var payload *gmail.MessagePart = msg.Payload

// }
