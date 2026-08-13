package storage

import (
	"database/sql"
	_ "embed"
	"log"
)

//go:embed schema.sql
var schema string

func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)

	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	result, err := db.Exec(schema)
	if err != nil {
		db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	log.Printf("Setup query executed: %v", result)

	return db, nil
}
