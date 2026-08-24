#!/bin/bash
set -e
DEST=/home/pi/open-gallery

mkdir -p "$DEST/deploy" "$DEST/data" "$DEST/temp_images"

chmod +x "$DEST/open-gallery"
chmod +x "$DEST/deploy/kiosk.sh"

# Install systemd services
sudo cp "$DEST/deploy/open-gallery.service" /etc/systemd/system/
sudo cp "$DEST/deploy/open-gallery-kiosk.service" /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now open-gallery
sudo systemctl enable open-gallery-kiosk

echo "Installed. Reboot to launch kiosk."
