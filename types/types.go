package types

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

/* Submission represents a single submission from any of the "SourceSystem."
 * The fields contain references to image source locations and submission related metadata.
 */
type Submission struct {
	SubmissionAuthor         string
	SubmissionContactDetails string
	SourceSystem             SourceSystem
	ImageLocation            string
	SubmissionConnection     string
	SubmissionStatus         SubmissionStatus
}

/* Source System Represents the source for the Submission structs.
 *
 * Is a constant int ranging 0 - 3 which formats as strings:
 * Unknown, Web, Email, and SMS respectively.
 */

type SourceSystem int

const (
	SourceUnknown SourceSystem = iota
	SourceWeb                  // Recall that QR code routes to webapp
	SourceEmail
	SourceSms
)

func (s SourceSystem) String() string {
	switch s {
	case SourceUnknown:
		return "Unknown"
	case SourceWeb:
		return "Web"
	case SourceEmail:
		return "Email"
	case SourceSms:
		return "SMS"
	}
	return "Unknown"
}

/* SubmissionStatus is an enum representing the status states
 *
 * Is a const int ranging from 0 - 4 which formats as strings
 * Unknown Failure, Moderation Pending, Moderation Rejected,
 * Moderation Accepted, and Complete respectively
 *
 */

type SubmissionStatus int

const (
	SubmissionUnkownFailure SubmissionStatus = iota
	SubmissionModerationPending
	SubmissionModerationRejected
	SubmissionModerationAccepted
	SubmissionComplete
)

func (s SubmissionStatus) String() string {
	switch s {
	case SubmissionUnkownFailure:
		return "Unknown Failure"
	case SubmissionModerationPending:
		return "Moderation Pending"
	case SubmissionModerationRejected:
		return "Moderation Rejected"
	case SubmissionModerationAccepted:
		return "Moderation Accepted"
	case SubmissionComplete:
		return "Complete"
	}
	return "Unknown Failure"
}

type GmailNotification struct {
	EmailAddress string `json:"emailAddress"`
	HistoryId    uint64 `json:"historyId"`
}

type CursorStore struct {
	mu   sync.Mutex
	path string
	id   uint64
}

func (c *CursorStore) Get() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.id }

func (c *CursorStore) Set(id uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id = id
	return os.WriteFile(c.path, []byte(strconv.FormatUint(id, 10)), 0600)
}

func NewCursorStore(path string) (*CursorStore, error) {
	c := &CursorStore{path: path}

	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil // first run, id stays 0
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("corrupt cursor file %s: %w", path, err)
	}
	c.id = id
	return c, nil
}
