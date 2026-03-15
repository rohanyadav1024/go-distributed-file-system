#!/bin/sh
set -eu

CA_DIR="${DFS_CA_DIR:-/ca}"
CERT_DIR="${DFS_CERT_DIR:-/certs}"
NODE_IP="$(hostname -i | awk '{print $1}')"
NODE_ID="${DFS_STORAGE_NODE_ID:-${HOSTNAME:-storaged}}"

export DFS_STORAGE_NODE_ID="$NODE_ID"
export DFS_STORAGE_LISTEN_ADDR="${DFS_STORAGE_LISTEN_ADDR:-${NODE_IP}:50052}"

mkdir -p "$CERT_DIR"
/opt/dfs/scripts/generate-certs.sh sign \
  "$CERT_DIR" \
  "$NODE_ID" \
  "DNS:${NODE_ID},DNS:storaged,IP:${NODE_IP}" \
  "$CA_DIR"

exec /usr/local/bin/storaged
