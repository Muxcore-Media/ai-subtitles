# AGENTS.md — ai-subtitles

MuxCore sidecar module (`ai-subtitles`).

## Module identity

| Field | Value |
|-------|-------|
| Directory | `ai-subtitles` |
| Role | `ai` |
| Capability | `ai.subtitles` |

## Build

```bash
cd ai-subtitles
go test ./...
make lint
```

## Agent rules

- Modules run as gRPC sidecars; capabilities are the security boundary.
- TLS required in production (`MUXCORE_INSECURE_DISABLE_TLS` is dev-only).
- Match existing Go patterns; run `gofmt` and package tests before finishing.
