#!/bin/sh
set -eu

usage() {
  cat <<USAGE
Usage:
  scripts/generate-certs.sh init [output_dir]
  scripts/generate-certs.sh sign <output_dir> <common_name> <subject_alt_name> <ca_dir>

Examples:
  scripts/generate-certs.sh init certs
  scripts/generate-certs.sh sign /certs metad 'DNS:metad,DNS:localhost,IP:127.0.0.1' /ca
USAGE
}

require_openssl() {
  if ! command -v openssl >/dev/null 2>&1; then
    echo "openssl is required" >&2
    exit 1
  fi
}

init_ca() {
  out_dir="$1"
  ca_dir="$out_dir/ca"

  mkdir -p "$ca_dir"

  if [ ! -f "$ca_dir/ca.key" ] || [ ! -f "$ca_dir/ca.crt" ]; then
    openssl genrsa -out "$ca_dir/ca.key" 4096
    openssl req -x509 -new -nodes \
      -key "$ca_dir/ca.key" \
      -sha256 -days "${DFS_CA_DAYS:-3650}" \
      -out "$ca_dir/ca.crt" \
      -subj "/CN=${DFS_CA_CN:-dfs-root-ca}"
  fi
}

sign_cert() {
  out_dir="$1"
  common_name="$2"
  san_list="$3"
  ca_dir="$4"

  mkdir -p "$out_dir"

  san_file="$out_dir/san.ext"
  csr_file="$out_dir/server.csr"
  # CA is mounted read-only in containers; keep serial file in writable cert dir.
  serial_file="$out_dir/ca.srl"

  printf 'subjectAltName=%s\n' "$san_list" > "$san_file"

  openssl genrsa -out "$out_dir/server.key" 2048
  openssl req -new \
    -key "$out_dir/server.key" \
    -out "$csr_file" \
    -subj "/CN=$common_name"
  openssl x509 -req \
    -in "$csr_file" \
    -CA "$ca_dir/ca.crt" \
    -CAkey "$ca_dir/ca.key" \
    -CAcreateserial \
    -CAserial "$serial_file" \
    -out "$out_dir/server.crt" \
    -days "${DFS_CERT_DAYS:-365}" \
    -sha256 \
    -extfile "$san_file"

  rm -f "$csr_file"
}

require_openssl

case "${1:-}" in
  init)
    init_ca "${2:-certs}"
    ;;
  sign)
    if [ "$#" -ne 5 ]; then
      usage
      exit 1
    fi
    sign_cert "$2" "$3" "$4" "$5"
    ;;
  *)
    usage
    exit 1
    ;;
esac
