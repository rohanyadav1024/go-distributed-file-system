# DFS

DFS is a Go-based distributed file system prototype with:

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
docker compose up --build --scale storaged=3
```

3. Available endpoints:

- `metad` gRPC: `localhost:50051`
- `metad` metrics: `localhost:9090`
- Prometheus: `localhost:9092`

## Deployment

Run from the repository root:

```bash
docker compose up --build --scale storaged=3
```

Or use the public deployment assets directly:

```bash
docker compose -f deploy/docker-compose.yml up --build --scale storaged=3
```

## Repository Layout

```text
cmd/        Go service entrypoints
internal/   Application packages
deploy/     Docker Compose and Prometheus assets
examples/   Example environment files
scripts/    Certificate and container entrypoint helpers
certs/      Generated local CA material
```

## Documentation

- [Phase 1 Overview](./dfs_phase1_README.md)
- [Architecture Diagram](./docs/architecture-diagram.md)
- [TLS Artifact Notes](./certs/README.md)

## Notes

- Example environment files live in `examples/`.
- Runtime certificates are generated from the local CA in `certs/`.
- The deployment supports `docker compose up --scale storaged=3`.
