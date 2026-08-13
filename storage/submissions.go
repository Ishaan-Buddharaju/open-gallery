package storage

import (
	"database/sql"

	"github.com/Ishaan-Buddharaju/open-gallery/types"
	_ "modernc.org/sqlite"
)

func SubmitGmailNotification(db *sql.DB, notification types.GmailNotification) (sql.Result, error) {
	result, err := db.Exec("INSERT INTO EMAIL_NOTIFICATIONS (email, recieved_timestamp, historyID) VALUES (?, ?, ?)",
		notification.EmailAddress, notification.ReceivedAt, notification.HistoryId)
	if err != nil {
		return nil, err
	}
	return result, nil
}
