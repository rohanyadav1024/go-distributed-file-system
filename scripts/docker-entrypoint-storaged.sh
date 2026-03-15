#!/bin/sh
set -eu

CA_DIR="${DFS_CA_DIR:-/ca}"
CERT_DIR="${DFS_CERT_DIR:-/certs}"
NODE_IP="$(hostname -i | awk '{print $1}')"
NODE_ID="${DFS_STORAGE_NODE_ID:-${HOSTNAME:-storaged}}"
SAFE_NODE_ID="$(printf '%s' "$NODE_ID" | tr '/:' '__')"
SERVICE_CERT_DIR="${CERT_DIR}/storaged/${SAFE_NODE_ID}"

export DFS_STORAGE_NODE_ID="$NODE_ID"
export DFS_STORAGE_LISTEN_ADDR="${DFS_STORAGE_LISTEN_ADDR:-${NODE_IP}:50052}"

mkdir -p "$CERT_DIR" "$SERVICE_CERT_DIR"
/opt/dfs/scripts/generate-certs.sh sign \
  "$SERVICE_CERT_DIR" \
  "$NODE_ID" \
  "DNS:${NODE_ID},DNS:storaged,IP:${NODE_IP}" \
  "$CA_DIR"

# Keep app paths stable while storing generated certs in service-specific dirs.
ln -sf "$CA_DIR/ca.crt" "$CERT_DIR/ca.crt"
ln -sf "$SERVICE_CERT_DIR/server.crt" "$CERT_DIR/server.crt"
ln -sf "$SERVICE_CERT_DIR/server.key" "$CERT_DIR/server.key"

exec /usr/local/bin/storaged
