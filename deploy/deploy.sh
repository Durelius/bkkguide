#!/usr/bin/env bash
# Build for the server, upload, swap the binary and restart. Usage: make deploy
set -euo pipefail
cd "$(dirname "$0")/.."

HOST=${DEPLOY_HOST:-wilhelm@172.232.147.147}
KEY=${DEPLOY_KEY:-$HOME/.ssh/id_ed25519_nopass}
SSH=(ssh -i "$KEY" -o BatchMode=yes "$HOST")

make build GOOS=linux GOARCH=amd64
scp -q -i "$KEY" bin/bkkguide "$HOST:/tmp/bkkguide.new"
"${SSH[@]}" '
  set -e
  sudo install -m 0755 /tmp/bkkguide.new /usr/local/bin/bkkguide
  rm -f /tmp/bkkguide.new
  sudo systemctl restart bkkguide
  sleep 1
  systemctl is-active --quiet bkkguide || { sudo journalctl -u bkkguide -n 20 --no-pager; exit 1; }
'
curl -sf -o /dev/null https://bkkguide.wilhelm.my/api/config && echo "Deployed: https://bkkguide.wilhelm.my"
