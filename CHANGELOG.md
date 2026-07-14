# Changelog

## Unreleased

### Added
- SSRF protection: block localhost, private/link-local IPs, `.local` hosts (`fetcher.ErrBlockedURL`)
- `GLEANER_BLOCK_PRIVATE_URLS` env (default `true` for server via `OptionsFromEnv`)
- Large-page golden fixture `testdata/golden_large_article.html` + integration test
- Benchmarks: `BenchmarkGetBlocks`, `BenchmarkGetBlockScore`, `BenchmarkSummarize`, `BenchmarkGetArticleSync`
- `Dockerfile.gse` — pure-Go image with GSE segmenter
- CI job `test-gse` (CGO_ENABLED=0 matrix)
- Server integration tests: HTTP errors, body limit, timeout, blocked URL
- HTTP `POST /` accepts `html` field in addition to `url`
- `GET /healthz` health check endpoint
- Structured JSON errors (`{"error":"..."}`) with HTTP status codes
- `cmd/gleaner` CLI (URL or HTML file input)
- `GLEANER_SEGMENT_BACKEND` environment variable (`gojieba` / `gse`)
- Fetcher `GetContext`, `MaxBodyBytes` (default 10 MiB)
- GitHub Actions CI workflow
- Golden integration tests with `testdata/golden_expected.json`

### Changed
- `GetBlockScore` uses UTF-8 rune count (matches JavaScript `String.length` for Chinese)
- `Article.Sentiment` is `int` (was `float64`)
- `DefaultClient().Close()` is a no-op to protect the package singleton
- HTTP server: graceful shutdown on SIGINT/SIGTERM

### Removed
- Redundant root `data/` directory (runtime uses `internal/data` via `go:embed`)

### Fixed
- Fetcher response body size limit prevents OOM on large pages
- `GetArticleSyncWithOptions` now applies `ResolveExpectedLength` and min/max defaults (was ignoring per-call length options)
- `ExtractBody` strips script/iframe before matching `<body>` (avoids nested iframe `body` false match)
- Fetcher `BlockPrivateURLs` flag now respected when toggled after `NewClient()`

## Initial release

- Go port of [ahkimkoo/arex](https://github.com/ahkimkoo/arex): article extraction, summarization, sentiment analysis, HTTP service
