# Raspberry Pi Setup

Steps to get Open Gallery running as a kiosk display on a Raspberry Pi: the
Go server runs as a background service and Chromium auto-launches
full-screen on boot to show the gallery.

These steps assume the Pi's user account is `ibuddhar` (current Raspberry Pi
Imager no longer creates a default `pi` user — whatever username you set
during imaging is your real account). Swap in your own username/home dir if
it differs, including in the `deploy/*.service` files and `install.sh`,
which hardcode `User=ibuddhar` / `/home/ibuddhar/open-gallery`.

## 0. What you'll need

- A Raspberry Pi running Raspberry Pi OS (Bookworm or newer), with your
  user account available and SSH enabled
- A monitor/TV connected to the Pi (the kiosk draws to `/dev/tty1` via X)
- Network access on the Pi (ingestion depends on reaching Gmail/Pub/Sub)
- `credentials.json` for the Gmail OAuth client, and a completed
  `token.json` (see step 4 below — OAuth consent needs a browser, so it's
  easiest to generate this on your dev machine first)
- A GCP service account JSON key for Pub/Sub (see step 5 below — separate
  from the Gmail OAuth creds above)
- Your real `.env` values (GCP project, topic, subscription, etc.)

## 1. Install OS packages on the Pi

```sh
sudo apt update
sudo apt install -y xinit xserver-xorg chromium-browser
```

`xinit` and `xserver-xorg` give the kiosk service a bare X session to run
Chromium in — no desktop environment is needed.

## 2. Build the binary

Open Gallery uses the pure-Go `modernc.org/sqlite` driver, so no CGO/cross
compiler is required. Build it on your dev machine for the Pi's
architecture (64-bit Raspberry Pi OS is `arm64`; older 32-bit images are
`arm` with `GOARM=7`):

```sh
GOOS=linux GOARCH=arm64 go build -o open-gallery .
```

Alternatively, install Go 1.26+ directly on the Pi and run
`go build -o open-gallery .` there.

## 3. Copy the project to the Pi

From your dev machine:

```sh
scp open-gallery ibuddhar@<pi-host>:/home/ibuddhar/open-gallery/
scp -r deploy ibuddhar@<pi-host>:/home/ibuddhar/open-gallery/
scp .env credentials.json token.json ibuddhar@<pi-host>:/home/ibuddhar/open-gallery/
```

`DB_PATH` and `IMAGE_PATH` directories (`data/`, `temp_images/` by default)
don't need to be copied — `deploy/install.sh` creates them.

## 4. Gmail OAuth token

`token.json` is produced by an interactive OAuth consent flow (it opens a
browser). The Pi is usually headless-ish (kiosk display, no desktop
browser), so generate the token on your dev machine first:

```sh
go run .   # completes the OAuth flow, writes token.json, then Ctrl+C
```

Then copy the resulting `token.json` to the Pi as shown in step 3. As long
as the token doesn't expire/get revoked, the Pi never needs to do its own
OAuth consent.

## 5. Pub/Sub service account credentials

This is a separate credential system from the Gmail OAuth creds above.
`ingest.SetupPubSubClient` (`ingest/email.go`) calls `pubsub.NewClient`
with no explicit credential option, so it authenticates via Google's
**Application Default Credentials (ADC)** — which looks for, in order, a
`GOOGLE_APPLICATION_CREDENTIALS` env var pointing to a key file, then a
`gcloud auth application-default login` cache, then (irrelevant here) GCE
metadata. A dev machine often has the `gcloud` cache already; a fresh Pi
has neither, and fails with:

```
fatal: pubsub client: failed to initialize Pub/Sub Client: ... could not find default credentials
```

Service account keys don't expire/rotate on their own (unlike OAuth
refresh tokens), which makes them the right fit for an unattended device.
Create one on your dev machine:

```sh
gcloud iam service-accounts create open-gallery-pubsub \
  --project=your-gcp-project \
  --display-name="Open Gallery Pub/Sub"

gcloud projects add-iam-policy-binding your-gcp-project \
  --member="serviceAccount:open-gallery-pubsub@your-gcp-project.iam.gserviceaccount.com" \
  --role="roles/pubsub.editor"

gcloud iam service-accounts keys create pubsub-key.json \
  --iam-account=open-gallery-pubsub@your-gcp-project.iam.gserviceaccount.com
```

(`pubsub.editor`, not a narrower publish/subscribe role, because
`SetupPubSubClient` also calls `CreateTopic`/`CreateSubscription` if they
don't already exist.)

Copy the key to the Pi and lock down its permissions:

```sh
scp pubsub-key.json ibuddhar@<pi-host>:/home/ibuddhar/open-gallery/
ssh ibuddhar@<pi-host> chmod 600 /home/ibuddhar/open-gallery/pubsub-key.json
```

Then add this line to `.env` (added in the next step):

```
GOOGLE_APPLICATION_CREDENTIALS=/home/ibuddhar/open-gallery/pubsub-key.json
```

No code change needed for this — `config.Load()` calls
`godotenv.Load()` unconditionally, which puts every `.env` line into the
real process environment, and Google's ADC resolution reads that env var
itself.

## 6. Configure `.env`

Make sure `/home/ibuddhar/open-gallery/.env` has your real values:

```sh
GCloudProjectID=your-gcp-project
GCloudTopicName=gmail-notifications
GCloudGmailSubscription=gmail-sub

DB_PATH=data/opengallery.db
IMAGE_PATH=temp_images/

GOOGLE_APPLICATION_CREDENTIALS=/home/ibuddhar/open-gallery/pubsub-key.json
```

The app loads `.env` itself via `godotenv` (see `config/config.go`) relative
to its working directory, so the systemd unit doesn't need
`EnvironmentFile=` — just make sure `WorkingDirectory` in
`open-gallery.service` matches where `.env` actually lives
(`/home/ibuddhar/open-gallery` by default).

## 7. Run the install script

On the Pi:

```sh
cd /home/ibuddhar/open-gallery
sudo ./deploy/install.sh
```

This script:
- creates `deploy/`, `data/`, `temp_images/` under the install dir
- makes `open-gallery` and `deploy/kiosk.sh` executable
- copies `open-gallery.service` and `open-gallery-kiosk.service` into
  `/etc/systemd/system/`
- enables and starts `open-gallery.service` immediately
- enables (but doesn't start) `open-gallery-kiosk.service`

## 8. Reboot into the kiosk

```sh
sudo reboot
```

On boot: `open-gallery.service` starts the Go server, and
`open-gallery-kiosk.service` launches an X session on `tty1` running
Chromium in `--kiosk` mode pointed at `http://127.0.0.1:8080`.
`deploy/kiosk.sh` polls the server until it responds before starting
Chromium, so ordering issues on boot are self-healing.

## 9. Verify / troubleshoot

```sh
# server status + logs
sudo systemctl status open-gallery
journalctl -u open-gallery -f

# kiosk status + logs
sudo systemctl status open-gallery-kiosk
journalctl -u open-gallery-kiosk -f
```

Common issues:
- **Blank/black screen, no Chromium** — check `open-gallery-kiosk` logs;
  confirm `xinit`/`xserver-xorg` are installed and nothing else (like a
  desktop environment) is already bound to `tty1`.
- **Server won't start / "GCloudProjectID must be set"** — `.env` is
  missing or not in `WorkingDirectory`; re-check step 6.
- **"could not find default credentials"** — `GOOGLE_APPLICATION_CREDENTIALS`
  is missing from `.env`, the key file isn't on the Pi, or it's owned by
  `root` from a `sudo scp`/`sudo cp` instead of your account (the service
  runs as `User=ibuddhar` and needs read access); re-check step 5.
- **"unable to open database file (14)"** — same root-ownership issue as
  above but for `data/`/`temp_images/`: if `install.sh` was run with `sudo`,
  those dirs got created as `root`. Fix with
  `sudo chown -R ibuddhar:ibuddhar /home/ibuddhar/open-gallery`.
- **Gmail ingestion silent** — confirm the Pub/Sub topic/subscription in
  `.env` match what you watched when generating `token.json`, and that the
  Pi has outbound network access.

## Updating after a code change

```sh
# on dev machine
GOOS=linux GOARCH=arm64 go build -o open-gallery .
scp open-gallery ibuddhar@<pi-host>:/home/ibuddhar/open-gallery/

# on the Pi
sudo systemctl restart open-gallery
```

The kiosk service doesn't need restarting unless `kiosk.sh` or the service
unit itself changed — it just points Chromium at the running server.
