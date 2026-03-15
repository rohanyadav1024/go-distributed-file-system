#!/bin/sh
set -eu

CA_DIR="${DFS_CA_DIR:-/ca}"
CERT_DIR="${DFS_CERT_DIR:-/certs}"

mkdir -p "$CERT_DIR"
/opt/dfs/scripts/generate-certs.sh sign \
  "$CERT_DIR" \
  "metad" \
  "DNS:metad,DNS:localhost,IP:127.0.0.1" \
  "$CA_DIR"

exec /usr/local/bin/metad
