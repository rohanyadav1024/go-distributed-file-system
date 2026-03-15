# TLS Artifacts

This directory is intentionally kept out of version control except for this placeholder.

Generate a local CA before running Docker:

```bash
./scripts/generate-certs.sh init certs
```

The Docker entrypoints generate runtime service certificates from `certs/ca/`.
