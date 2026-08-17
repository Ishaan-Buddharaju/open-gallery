package ingest

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/Ishaan-Buddharaju/open-gallery/storage"
	"github.com/Ishaan-Buddharaju/open-gallery/types"
	"golang.org/x/net/html"
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

func ReceiveGmailNotifications(ctx context.Context, subClient *pubsub.Subscriber, srv *gmail.Service, db *sql.DB) error {
	handler := func(ctx context.Context, msg *pubsub.Message) {
		var n types.GmailNotification
		if err := json.Unmarshal(msg.Data, &n); err != nil {
			log.Printf("unparseable payload %q: %v", msg.Data, err)

			msg.Ack()
			return
		}
		n.ReceivedAt = msg.PublishTime
		// _, err := storage.SubmitGmailNotification(ctx, db, n)
		// if err != nil {
		// 	log.Printf("SubmitGmailNotification failed: %v", err)
		// }

		log.Printf("Received notification (historyId=%d)", n.HistoryId)
		if err := process(ctx, srv, n, db); err != nil {
			log.Printf("process failed (historyId=%d): %v", n.HistoryId, err)
			msg.Nack()
			return
		}
		msg.Ack()
		log.Printf("acked notification (historyId=%d)", n.HistoryId)
	}
	log.Printf("Recieving Gmail notifications now")
	return subClient.Receive(ctx, handler)
}

func process(ctx context.Context, srv *gmail.Service, n types.GmailNotification, db *sql.DB) error {
	var cursor uint64
	var err error
	cursor, err = storage.GetCursor(db, "gmail")
	log.Printf("GetCursor in process: cursor=%d", cursor)
	if err != nil {
		log.Printf("GetCursor failed: %v", err)
		return err
	}

	newCursor, newMessages, err := listNewMessages(srv, cursor)
	if newCursor != n.HistoryId {
		log.Printf("Warning: newCursor (%d) != n.HistoryId (%d)", newCursor, n.HistoryId)
	}
	log.Printf("Found %d new messages in processing", len(newMessages))
	if err != nil {
		log.Printf("listNewMessages failed: %v", err)
		return err
	}

	err = processNewMessages(ctx, srv, newMessages, db)
	if err != nil {
		log.Printf("processNewMessages failed: %v", err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Begin failed: %v", err)
		return err
	}
	defer tx.Rollback()
	err = storage.UpdateCursor(tx, "gmail", newCursor)

	if err != nil {
		log.Printf("UpdateCursor failed: %v", err)
		return err
	}

	return tx.Commit()
}

/* Notification Processing after PubSub notification */

// TODO Add Oauth Device flow for the projector to setup the source inbox plus filters
func listNewMessages(gmailService *gmail.Service, startHistoryID uint64) (uint64, []*gmail.Message, error) {
	var req *gmail.UsersHistoryListCall = gmailService.Users.History.List("me").
		StartHistoryId(startHistoryID).
		HistoryTypes("messageAdded")
	result, err := req.Do()
	log.Printf("History result: %v", result)
	if err != nil {
		return 0, nil, err
	}
	var newMessages []*gmail.Message
	for _, h := range result.History {

		for _, m := range h.Messages {
			newMessages = append(newMessages, m)
			log.Printf("Found message %s", m.Id)
		}
	}
	return result.HistoryId, newMessages, nil

}

func processNewMessages(ctx context.Context, srv *gmail.Service, newMessages []*gmail.Message, db *sql.DB) error {
	for _, message := range newMessages {
		email, err := processMessage(ctx, srv, message)
		if err != nil {
			return err
		}
		log.Printf("Processed message %s", message.Id)
		err = storage.AddSubmission(db, email)
		if err != nil {
			log.Printf("Failed to add submission to db: %v", err)
			return err
		}
	}

	return nil
}

func processMessage(ctx context.Context, srv *gmail.Service, message *gmail.Message) (types.Submission, error) {
	msg, err := srv.Users.Messages.Get("me", message.Id).Format("full").Context(ctx).Do()
	email := types.Submission{}
	if err != nil {
		return email, err
	}
	parseMsgTree(ctx, srv, message.Id, msg.Payload, &email)
	from, err := parseEnvelope(msg.Payload, "From")
	if err != nil {
		log.Printf("Failed to parse From header: %v", err)
		return email, err
	}
	addr, err := mail.ParseAddress(from)
	if err != nil {
		log.Printf("Failed to parse From header: %v", err)
		return email, err
	}
	email.Author = addr.Name
	email.ContactDetails = addr.Address
	email.SourceSystem = types.SourceEmail
	tagsJSON, err := json.Marshal(recipientTags(msg.Payload))
	if err != nil {
		log.Printf("Failed to marshal tags: %v", err)
	}
	email.ConnectionTags = string(tagsJSON)
	email.Status = types.SubmissionModerationPending
	return email, nil
}

func parseMsgTree(ctx context.Context, srv *gmail.Service, msgID string, part *gmail.MessagePart, email *types.Submission) {
	if part == nil {
		return
	} else if part.MimeType == "text/plain" && email.Body == "" {
		raw, err := decodeBase64(part.Body.Data)
		if err != nil {
			log.Fatalf("Failed to decode text/plain part: %v", err)
		}
		text := string(raw)
		email.Body = text
		log.Printf("Found text/plain part: %s", text)
	} else if part.MimeType == "text/html" && email.Body == "" {
		raw, err := decodeBase64(part.Body.Data)
		if err != nil {
			log.Fatalf("Failed to decode text/plain part: %v", err)
		}
		text := string(raw)
		text = stripHTML(text)
		log.Printf("Found text/html part: %s", text)
		email.Body = text
	} else if part.MimeType == "text/plain" || part.MimeType == "text/html" {
		return // already processed in a different recursive call
	} else if part.MimeType == "multipart/alternative" {
		log.Printf("Found multipart/alternative part")
		for _, p := range part.Parts {
			parseMsgTree(ctx, srv, msgID, p, email)
		}
	} else if part.MimeType == "multipart/mixed" {
		log.Printf("Found multipart/mixed part")
		for _, p := range part.Parts {
			parseMsgTree(ctx, srv, msgID, p, email)
		}
	} else if part.MimeType == "multipart/related" {
		log.Printf("Found multipart/related part")
		for _, p := range part.Parts {
			parseMsgTree(ctx, srv, msgID, p, email)
		}
	} else if strings.HasPrefix(part.MimeType, "image/") {
		log.Printf("Found image part: %s", part.MimeType)
		imgPath, err := saveImage(ctx, srv, msgID, part, os.Getenv("IMAGE_PATH"))
		if err != nil {
			log.Printf("Failed to save image: %v", err)
		} else if imgPath != "" {
			var paths []string
			if email.ImagePaths != "" {
				if err := json.Unmarshal([]byte(email.ImagePaths), &paths); err != nil {
					log.Printf("Failed to unmarshal existing image paths: %v", err)
					return
				}
			}
			paths = append(paths, imgPath)
			encoded, err := json.Marshal(paths)
			if err != nil {
				log.Printf("Failed to marshal image paths: %v", err)
				return
			}
			email.ImagePaths = string(encoded)
			log.Printf("Saved image: %s", imgPath)
		}
	} else {
		log.Fatalf("Unsupported MIME type: %s", part.MimeType)
	}
}

func decodeBase64(data string) ([]byte, error) {
	raw, err := base64.URLEncoding.DecodeString(data)
	if err == nil {
		return raw, nil
	}
	return base64.RawURLEncoding.DecodeString(data)
}

func stripHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return strings.Join(strings.Fields(b.String()), " ")
}

func parseEnvelope(part *gmail.MessagePart, name string) (string, error) {
	for _, h := range part.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value, nil
		}
	}
	return "", fmt.Errorf("header not found: %s", name)
}

func saveImage(ctx context.Context, srv *gmail.Service, msgID string, part *gmail.MessagePart, dir string) (string, error) {
	// Only PNG and JPEG for now
	var ext string
	switch part.MimeType {
	case "image/jpeg", "image/jpg", "image/JPG", "image/JPEG":
		ext = ".jpg"
	case "image/png", "image/PNG":
		ext = ".png"
	default:
		log.Printf("Unsupported image/%s type, skipping... ", part.MimeType)
		return "", nil // skip
	}

	if part.Body.AttachmentId == "" {
		return "", nil
	}

	att, err := srv.Users.Messages.Attachments.
		Get("me", msgID, part.Body.AttachmentId).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("fetch attachment: %w", err)
	}
	data, err := decodeBase64(att.Data)
	if err != nil {
		return "", fmt.Errorf("decode attachment: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	final := filepath.Join(dir, fmt.Sprintf("%s-%s%s", msgID, part.PartId, ext))

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), final); err != nil {
		return "", err
	}
	return final, nil
}

func recipientTags(p *gmail.MessagePart) []string {
	seen := map[string]bool{}
	for _, h := range []string{"To", "Cc", "Delivered-To"} {
		raw, err := parseEnvelope(p, h)
		if err != nil {
			continue
		}
		list, err := mail.ParseAddressList(raw)
		if err != nil {
			continue
		}
		for _, a := range list {
			if t := parseTag(a.String()); t != "" {
				seen[t] = true
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

func parseTag(addr string) string {
	a, err := mail.ParseAddress(addr)
	if err != nil {
		return ""
	}
	local, _, found := strings.Cut(a.Address, "@")
	if !found {
		return ""
	}
	_, tag, found := strings.Cut(local, "+")
	if !found {
		return ""
	}
	return strings.ToLower(tag)
}
