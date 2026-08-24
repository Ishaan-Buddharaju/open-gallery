# Open Gallery

A self-hosted gallery wall service that can run on memory constrainted devices. People can send in photos via your email, sms, or website then they're injested into a single normalized pipeline, and a full-screen web view displays the accepted submissions as a dynamic, animated gallery wall. 

## How it works
* As of 8/24/26 SMS and Website submissions are not setup yet
```
Gmail ──▶ Pub/Sub notification ──▶ ingest ──▶ SQLite ──▶ web gallery
```

1. Gmail pushes a notification to a Pub/Sub topic when new mail arrives.
2. The ingest worker reads its stored cursor (`INGEST_CURSOR`), pulls the
   messages added since then via the Gmail History API, and walks the MIME tree.
3. Body text becomes the caption; image attachments are written to `IMAGE_PATH`.
4. The normalized record is inserted into `NORMALIZED_SUBMISSIONS` and the
   cursor is advanced in the same transaction.
5. The web server serves accepted submissions as a justified photo grid.

## Requirements

- Go 1.26+
- A Google Cloud project with a Pub/Sub topic and subscription
- Gmail OAuth client credentials (`credentials.json`)

No CGO required — SQLite is the pure-Go `modernc.org/sqlite` driver.

## Configuration

Create a `.env` file in the project root:

```sh
GCloudProjectID=your-gcp-project
GCloudTopicName=gmail-notifications
GCloudGmailSubscription=gmail-sub

DB_PATH=data/opengallery.db
IMAGE_PATH=temp_images/

# optional
CREDS_FILE=credentials.json
TOKEN_FILE=token.json
HTTP_ADDR=127.0.0.1:8080
```

`GCloudProjectID`, `DB_PATH`, and `IMAGE_PATH` are required; the rest have
defaults.

## Build & run

```sh
go build -o open-gallery .
./open-gallery
```

On first run a browser opens for the Gmail OAuth consent screen and the
resulting token is saved to `token.json`.

Then open http://127.0.0.1:8080.

## Endpoints

| Route | Description |
| --- | --- |
| `GET /` | Full-screen gallery view |
| `GET /harness` | Test harness page for layout work |
| `GET /api/submissions` | Accepted submissions as JSON |
| `GET /images/{name}` | Serves a stored image (jpg/jpeg/png only) |

## Layout

```
main.go        wiring: config, OAuth, DB, Pub/Sub, web server
config/        .env loading and validation
ingest/        Gmail push ingestion and MIME parsing
storage/       SQLite access, embedded schema, cursor helpers
types/         shared domain types (Submission, enums)
web/           HTTP server and embedded gallery frontend
deploy/        systemd units and Raspberry Pi kiosk install script
```

## Deploying to a Raspberry Pi

Copy the built binary plus the `deploy/` directory to `/home/pi/open-gallery`,
then:

```sh
./deploy/install.sh
```

This installs and enables the `open-gallery` service and the Chromium kiosk
unit. Reboot to launch the display.

## License

MIT — see [LICENSE](LICENSE).
