package injest

import (
	"fmt"
	"os"
	"context"
	"log"
	"net/http"
	"encoding/json"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		Name: subPath,
		Topic: topicPath,
		AckDeadlineSeconds: 30,
		RetainAckedMessages: false,
		EnableExactlyOnceDelivery: false,
	}
	_, err = client.SubscriptionAdminClient.CreateSubscription(ctx, pbSubscription)
	if err != nil && !IsAlreadyExists(err) {
		return nil, fmt.Errorf("subscription creation failed: %w", err)
	}

	sub := client.Subscriber(subPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize subscriber runtime: %w", err)
	}

	sub.ReceiveSettings.NumGoroutines = 4
	sub.ReceiveSettings.MaxOutstandingMessages = 20

	return sub, nil
}

func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.AlreadyExists
}

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

// TODO Add Oauth Device flow for the projector to setup the source inbox plus filters
func FetchInbox(gmailService *gmail.Service, query string) (*gmail.ListMessagesResponse, error) {
	var req *gmail.UsersMessagesListCall = gmailService.Users.Messages.List("me")
	if query != "" {
		req.Q(query)
	}

	return req.Do()
}
