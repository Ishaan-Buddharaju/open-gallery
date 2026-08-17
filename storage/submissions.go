package storage

import (
	"database/sql"
	"log"

	"github.com/Ishaan-Buddharaju/open-gallery/types"
	_ "modernc.org/sqlite"
)

func SeedCursor(db *sql.DB, historyID uint64, source string) error {
	query := "INSERT INTO INGEST_CURSOR (source, received_timestamp, history_id) VALUES (?1, CURRENT_TIMESTAMP, ?2) ON CONFLICT(source) DO NOTHING"
	_, err := db.Exec(query, source, historyID)
	return err
}

func GetCursor(db *sql.DB, source string) (uint64, error) {
	var cursor uint64
	query := "SELECT history_id FROM INGEST_CURSOR WHERE source = ?"
	err := db.QueryRow(query, source).Scan(&cursor)
	return cursor, err
}

func UpdateCursor(tx *sql.Tx, source string, historyID uint64) error {
	query := "UPDATE INGEST_CURSOR SET history_id = ?1 WHERE source = ?2 AND history_id < ?1"
	_, err := tx.Exec(query, historyID, source)
	log.Printf("updated cursor for source=%s to historyId=%d", source, historyID)
	return err
}

func AddSubmission(db *sql.DB, submission types.Submission) error {
	query := "INSERT INTO NORMALIZED_SUBMISSIONS (source, status, author, contact_info, attribution_tags, image_paths, caption) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)"
	_, err := db.Exec(query, submission.SourceSystem, submission.Status, submission.Author, submission.ContactDetails, submission.ConnectionTags, submission.ImagePaths, submission.Body)
	return err
}
