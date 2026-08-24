#!/bin/bash
xset s off
xset -dpms
xset s noblank

# Wait for the Go server to be ready
until curl -sf http://127.0.0.1:8080/ > /dev/null 2>&1; do
  sleep 1
done

exec chromium-browser \
  --kiosk \
  --noerrdialogs \
  --disable-infobars \
  --disable-session-crashed-bubble \
  --incognito \
  --disable-translate \
  --disable-gpu-compositing \
  http://127.0.0.1:8080
