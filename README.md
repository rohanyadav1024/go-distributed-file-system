# 1024-dfs

## 1024

1024 is a secure, observable, self-healing distributed storage system...

1024-dfs is a Go-based distributed storage prototype with:

- `metad` for metadata management
- scalable `storaged` nodes for chunk storage
- Prometheus metrics for observability

## Quick Start

1. Generate a local CA:

```bash
./scripts/generate-certs.sh init certs
```

2. Start the stack:

```bash
docker compose -f deploy/docker-compose.yml up --build --scale storaged=3
```

3. Available endpoints:

- `metad` gRPC: `localhost:50051`
- `metad` metrics: `localhost:9090`
- Prometheus: `localhost:9092`

## Deployment

Run from the repository root:

```bash
docker compose -f deploy/docker-compose.yml up --build --scale storaged=3
```

Or run from the `deploy/` directory:

```bash
cd deploy
docker compose up --build --scale storaged=3
```

## Local Run (Go CLI)

The binaries currently read TLS files from absolute paths:

- `/certs/ca.crt`
- `/certs/server.crt`
- `/certs/server.key`

Step-by-step local flow (single-node):

```bash
./scripts/generate-certs.sh init certs
./scripts/generate-certs.sh sign certs/metad metad 'DNS:metad,DNS:localhost,IP:127.0.0.1' certs/ca
sudo mkdir -p /certs
sudo cp certs/ca/ca.crt /certs/ca.crt
sudo cp certs/metad/server.crt /certs/server.crt
sudo cp certs/metad/server.key /certs/server.key
DFS_JWT_SECRET=change-me go run ./cmd/metad
```

In a second terminal:

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

Optional token helper:

```bash
DFS_JWT_SECRET=change-me go run ./cmd/token
```

## Repository Layout

```text
cmd/        Go service entrypoints
internal/   Application packages
deploy/     Docker Compose and Prometheus assets
deploy/docker/ Dockerfiles for deploy targets
examples/   Example environment files
scripts/    Certificate and container entrypoint helpers
certs/      Generated local CA material
```

## Documentation

- [Phase 1 Overview](./docs/phase1.md)
- [Architecture Diagram](./docs/architecture-diagram.md)
- [TLS Artifact Notes](./certs/README.md)

## Notes

- Example environment files live in `examples/`.
- Runtime certificates are generated from the local CA in `certs/`.
- The deployment supports `docker compose -f deploy/docker-compose.yml up --scale storaged=3`.
- Docker is the recommended path for multi-node runs.
