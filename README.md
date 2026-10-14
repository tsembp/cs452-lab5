# CS452 Lab 5 — Web Search

A Go HTTP document-search server with a static web interface and JSON search API.

## Run

```bash
go build ./cmd/...
./webserver -addr :8080 -index ./test/data/invertedindex-small.txt -topk 3
```

Open `http://localhost:8080`, or query the API directly:

```bash
curl 'http://localhost:8080/api/search?q=legacy%20adventure'
```

## Implemented

- `GET /` serves the search page.
- `GET /api/search?q=...` normalizes query terms and returns ranked JSON results.
- Top-K result limiting.
- Inverted-index preloading at server startup.
- In-memory HTML caching.

## Benchmark results

All runs used Vegeta for 30 seconds at a target rate of 10,000 requests/second with 100 maximum workers.

| Configuration | Endpoint | Throughput (req/s) | Mean latency | p99 latency | Mean response | Success |
|---|---|---:|---:|---:|---:|---:|
| File-scan baseline | `/api/search?q=legacy adventure` | 4,853.10 | 20.455 ms | 98.001 ms | 604 B | 100% |
| Preloaded index, Top-K = 3 | `/api/search?q=legacy adventure` | 9,999.92 | 0.503 ms | 2.805 ms | 198 B | 100% |
| HTML from file | `/` | 9,999.89 | 5.207 ms | 16.266 ms | 1050 B | 100% |
| Cached HTML | `/` | 9,999.98 | 0.502 ms | 3.229 ms | 1050 B | 100% |