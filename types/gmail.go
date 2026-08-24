package types

import (
	"time"
)

type GmailNotification struct {
	EmailAddress string    `json:"emailAddress"`
	HistoryId    uint64    `json:"historyId"`
	ReceivedAt   time.Time `json:"timestamp"`
}
