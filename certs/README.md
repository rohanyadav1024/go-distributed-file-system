# 1024-dfs TLS Artifacts

This directory is intentionally kept out of version control except for this placeholder.

Generate a local CA before running Docker:

```bash
./scripts/generate-certs.sh init certs
```

Docker runtime certificate layout:

- `certs/ca/`: local CA (`ca.crt`, `ca.key`)
- `certs/metad/`: generated metadata service certificate
- `certs/storaged/<node-id>/`: generated storage-node certificate per replica

The entrypoints also create compatibility symlinks so existing binaries can keep reading:

- `/certs/ca.crt`
- `/certs/server.crt`
- `/certs/server.key`

For host `go run`, create the same absolute files manually:

```bash
sudo mkdir -p /certs
sudo cp certs/ca/ca.crt /certs/ca.crt
```

Then copy the active service cert/key to:

- `/certs/server.crt`
- `/certs/server.key`
