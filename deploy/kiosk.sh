#!/bin/bash
xset s off
xset -dpms
xset s noblank

# Wait for the Go server to be ready
until curl -sf http://127.0.0.1:8080/ > /dev/null 2>&1; do
  sleep 1
done

CHROMIUM=$(command -v chromium-browser || command -v chromium)

RES=$(xrandr --query | awk '/\*/ {print $1; exit}')
WIDTH=${RES%x*}
HEIGHT=${RES#*x}

exec "$CHROMIUM" \
  --kiosk \
  --window-position=0,0 \
  --window-size="$WIDTH,$HEIGHT" \
  --start-fullscreen \
  --noerrdialogs \
  --disable-infobars \
  --disable-session-crashed-bubble \
  --incognito \
  --disable-translate \
  http://127.0.0.1:8080
