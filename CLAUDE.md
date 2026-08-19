# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Open Gallery is a Go application that ingests image submissions from multiple source channels (email, web, SMS) into a unified pipeline for moderation and display. Currently, the Gmail/email ingestion path is the active focus.

## Build & Run

```bash
go build -o open-gallery .
go run .
```

The app requires `credentials.json` and `token.json` for Gmail OAuth, plus a `.env` file with GCloud and DB config. On first run, it opens a browser for OAuth consent and saves the token.

## Architecture

**Ingestion pipeline (push-based):** Gmail → Google Pub/Sub notification → `ingest.ReceiveGmailNotifications` → fetch new messages via Gmail History API → parse MIME tree → save to SQLite.

Key flow in `ingest/email.go`:
1. Pub/Sub delivers a `GmailNotification` with a `historyId`
2. `process()` reads the stored cursor from `INGEST_CURSOR`, calls `listNewMessages` to get messages added since that cursor
3. `processNewMessages` iterates messages, calling `processMessage` which recursively walks the MIME tree (`parseMsgTree`) to extract body text and image attachments
4. Images are saved to disk (`IMAGE_PATH` env var, default `temp_images/`)
5. The normalized `Submission` is inserted into `NORMALIZED_SUBMISSIONS`
6. The cursor is advanced in a transaction

**Storage (`storage/`):** SQLite via `modernc.org/sqlite` (pure-Go, no CGO). Schema is embedded via `//go:embed schema.sql`. Single-writer (`MaxOpenConns=1`), WAL mode. Two tables: `INGEST_CURSOR` (tracks last-processed historyId per source) and `NORMALIZED_SUBMISSIONS` (unified submission records from all sources).

**Types (`types/`):** Shared domain types. `SourceSystem` and `SubmissionStatus` are iota enums with `String()` methods. `Submission` is the normalized record written to the DB.

**Planned but stubbed:** `server/` (display/API), `moderation-service/`, `ingest/.sms.go`, `ingest/.web.go` (dot-prefixed to exclude from build).

## Environment Variables (`.env`)

- `GCloudProjectID`, `GCloudTopicName`, `GCloudGmailSubscription` — Pub/Sub config
- `DB_PATH` — SQLite database path (default: `data/opengallery.db`)
- `IMAGE_PATH` — directory for downloaded image attachments

## Conventions

- Dot-prefixed `.go` files (e.g., `.sms.go`, `.web.go`) are used to keep planned-but-unimplemented source files out of the build.
- JSON-encoded arrays are stored in TEXT columns (`image_paths`, `attribution_tags`).
- The cursor pattern (`SeedCursor`/`GetCursor`/`UpdateCursor`) ensures at-least-once delivery with idempotent cursor advancement (`WHERE history_id < ?`).
