#!/usr/bin/env bash
# Create a BKK Guide admin on the server, as the service user that owns the
# database. Install as /usr/local/bin/bkkguide-create-admin and run:
#   sudo /usr/local/bin/bkkguide-create-admin   (CentOS sudo leaves /usr/local/bin off its PATH)
set -euo pipefail
exec sudo -u bkkguide env BKK_DB=/var/lib/bkkguide/bkkguide.db /usr/local/bin/bkkguide -create-admin
