# 1024-dfs Phase 1

## Index

- [Description](#description)
- [Quick Start](#quick-start)
- [Run With Docker](#run-with-docker)
- [Run With Go CLI](#run-with-go-cli)
- [Project Structure](#project-structure)
- [Upcoming](#upcoming)

## Description

1024-dfs is a Go-based distributed storage prototype.

Phase 1 is centered on two services:

- `metad`: metadata service backed by SQLite
- `storaged`: storage node service backed by the local chunk store

What Phase 1 currently includes:

- metadata RPCs for prepare, commit, get, list, and delete file metadata
- chunk RPCs for put, get, delete, and health checks
- storage node heartbeats and healthy-node tracking
- replication-aware placement
- background repair for under-replicated chunks
- Prometheus metrics for metadata and storage services
- JWT auth on metadata RPCs
- mTLS between services

## Quick Start

Recommended path:

```bash
./scripts/generate-certs.sh init certs
docker compose -f deploy/docker-compose.yml up --build --scale storaged=3
```

This starts:

- `metad` on `50051`
- `metad` metrics on `9090`
- scaled `storaged` replicas on the internal Docker network
- Prometheus on `9092`

Stop the stack with:

```bash
docker compose down
```

## Run With Docker

Use the canonical deploy compose file:

```bash
docker compose -f deploy/docker-compose.yml up --build --scale storaged=3
```

Notes:

- generate the local CA first with `./scripts/generate-certs.sh init certs`
- runtime service certificates are created by the Docker entrypoint scripts
- Prometheus configuration lives in `deploy/prometheus.yml`
- the deployment no longer hardcodes `storaged1`, `storaged2`, or `storaged3`

## Run With Go CLI

Requirements:

- Go `1.24.5`
- locally available TLS files at `/certs/ca.crt`, `/certs/server.crt`, and `/certs/server.key`

Generate local certs and prepare `/certs`:

```bash
./scripts/generate-certs.sh init certs
./scripts/generate-certs.sh sign certs/metad metad 'DNS:metad,DNS:localhost,IP:127.0.0.1' certs/ca
sudo mkdir -p /certs
sudo cp certs/ca/ca.crt /certs/ca.crt
sudo cp certs/metad/server.crt /certs/server.crt
sudo cp certs/metad/server.key /certs/server.key
```

Run `metad`:

```bash
DFS_JWT_SECRET=change-me go run ./cmd/metad
```

Run a storage node in another terminal:

```bash
./scripts/generate-certs.sh sign certs/storaged/local-node local-node 'DNS:local-node,DNS:localhost,IP:127.0.0.1' certs/ca
sudo cp certs/storaged/local-node/server.crt /certs/server.crt
sudo cp certs/storaged/local-node/server.key /certs/server.key
DFS_STORAGE_NODE_ID=local-node \
DFS_STORAGE_LISTEN_ADDR=127.0.0.1:50052 \
DFS_STORAGE_DATA_PATH=./data/local-node \
DFS_STORAGE_CAPACITY_BYTES=10737418240 \
DFS_METADATA_ADDR=127.0.0.1:50051 \
go run ./cmd/storaged
```

Optional helper for metadata RPC testing:

```bash
DFS_JWT_SECRET=change-me go run ./cmd/token
```

CLI note:

- Docker is the easiest way to run a full multi-node Phase 1 setup because certificate generation and service wiring are handled there
- local `go run` is best for single-node development with explicit cert setup

## Project Structure

```text
1024-dfs/
├── cmd/
│   ├── dfsctl/
│   ├── metad/
│   ├── storaged/
│   └── token/
├── certs/
├── deploy/
├── docs/
├── examples/
├── internal/
│   ├── auth/
│   ├── common/
│   ├── constants/
│   ├── metadata/
│   ├── metrics/
│   ├── node/
│   ├── observability/
│   ├── protocol/
│   ├── security/
│   └── storage/
├── scripts/
├── test/
├── deploy/docker-compose.yml
├── Dockerfile.metad
├── Dockerfile.storaged
├── README.md
└── docs/phase1.md
```

## Upcoming

- parameterize TLS paths for simpler non-Docker local runs
- fill `cmd/dfsctl/` with a usable developer CLI
- expand end-to-end coverage in `test/`
- complete currently partial metadata response fields such as replica addresses and chunk offsets
- grow the placeholder `internal/observability/` and `internal/security/` areas in later phases
