# AI Subtitles

Generates sidecar subtitles when `media-subtitles` catalog search fails, and aligns subtitle files downloaded from a different source than the media.

Does **not** replace `media-subtitles`. That module stays catalog-only and emits `media.subtitles.search.failed`.

## Ports

| Service | Default |
|---------|---------|
| gRPC / mesh | `127.0.0.1:9762` |
| Health / HTTP | `127.0.0.1:9763` |

## HTTP

| Method | Path | Notes |
|--------|------|-------|
| POST | `/v1/generate` | ASR/heuristic cues → SRT |
| POST | `/v1/sync` | offset + optional duration scale |
| POST | `/v1/from-search-failed` | consume `SubtitleSearchFailedPayload` |
| GET | `/v1/jobs` | recent generate/sync jobs |

## Privacy

Generated cues and file paths stay local. Transcripts are operational records with a delete path (`GET /v1/jobs` is in-memory until restart unless persisted by the operator).
