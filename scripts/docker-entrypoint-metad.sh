#!/bin/sh
set -eu

CA_DIR="${DFS_CA_DIR:-/ca}"
CERT_DIR="${DFS_CERT_DIR:-/certs}"
SERVICE_CERT_DIR="${CERT_DIR}/metad"

mkdir -p "$CERT_DIR" "$SERVICE_CERT_DIR"
/opt/dfs/scripts/generate-certs.sh sign \
  "$SERVICE_CERT_DIR" \
  "metad" \
  "DNS:metad,DNS:localhost,IP:127.0.0.1" \
  "$CA_DIR"

# Keep app paths stable while storing generated certs in service-specific dirs.
ln -sf "$CA_DIR/ca.crt" "$CERT_DIR/ca.crt"
ln -sf "$SERVICE_CERT_DIR/server.crt" "$CERT_DIR/server.crt"
ln -sf "$SERVICE_CERT_DIR/server.key" "$CERT_DIR/server.key"

exec /usr/local/bin/metad
