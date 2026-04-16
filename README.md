# speedtest-exporter

A Prometheus exporter that runs a speedtest on each scrape and exposes the results as metrics.
Useful for monitoring internet connection performance over time.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9090` | Port to listen on |
| `PING_MODE` | `tcp` | Latency measurement method. See [Ping modes](#ping-modes) below. |
| `SERVER_ID` | `-1` | Speedtest server ID to use. `-1` picks the closest server automatically. |
| `SERVER_FALLBACK` | `false` | If the configured server is unavailable, fall back to the closest one. |
| `TIMEOUT` | `60` | Maximum time (in seconds) to wait for a speedtest to complete. |

## Ping modes

The `PING_MODE` setting controls how latency is measured:

- **`tcp`** (default) - Uses TCP packets. Works everywhere, no special permissions needed.
- **`http`** - Measures latency via HTTP requests. Good for testing HTTP path latency specifically.
- **`icmp`** - Uses ICMP echo (traditional "ping"). **Requires root or `CAP_NET_RAW` capability** because raw ICMP sockets need elevated privileges.

To use ICMP mode with Docker:
```bash
docker run --rm -p 9090:9090 --cap-add=NET_RAW speedtest-exporter
```

## Metrics

The exporter exposes these Prometheus metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `speedtest_up` | gauge | Test succeeded (1) or failed (0) |
| `speedtest_scrape_duration_seconds` | gauge | How long the speedtest took to run |
| `speedtest_latency_seconds` | gauge | Average latency during the test |
| `speedtest_latency_min_seconds` | gauge | Best (lowest) latency recorded |
| `speedtest_latency_max_seconds` | gauge | Worst (highest) latency recorded |
| `speedtest_latency_jitter_seconds` | gauge | Latency variation |
| `speedtest_download_bytes_per_second` | gauge | Download speed |
| `speedtest_upload_bytes_per_second` | gauge | Upload speed |
| `speedtest_server_distance_kilometers` | gauge | Distance to the test server in km. Label: `server_id`. |
| `speedtest_server_info` | gauge | Server details (always 1). Labels: `server_id`, `server_name`, `server_country`, `user_isp`, `server_lat`, `server_lon`. |

## Quick start

### Run locally

```bash
go run ./cmd/speedtest-exporter

# or build and run:
go build -o speedtest-exporter ./cmd/speedtest-exporter
./speedtest-exporter
```
Then test it:

```bash
curl http://localhost:9090/metrics
```
### Docker

Build the image:
```bash
docker build -t speedtest-exporter .
```

Run it:
```bash
docker run --rm -p 9090:9090 speedtest-exporter
```

With custom settings:
```bash
docker run --rm -p 9090:9090 -e PING_MODE=icmp \
  --cap-add=NET_RAW speedtest-exporter
```

## Prometheus config

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'speedtest'
    static_configs:
      # ideally resolve the host via an internal docker network, dont expose the exporter
      - targets: ['monitoring_speedtest:9090']
    # speedtests can take a while and can be invasive to the network stability, so set a long scrape interval
    scrape_interval: 30m
```

## Sample output

```
speedtest_up 1
speedtest_scrape_duration_seconds 23.057467083
speedtest_download_bytes_per_second 5.764283915641032e+07
speedtest_upload_bytes_per_second 2.216889469920968e+06
speedtest_latency_seconds 0.0211362
speedtest_latency_min_seconds 0.014136
speedtest_latency_max_seconds 0.0394256
speedtest_latency_jitter_seconds 0.008914978
speedtest_server_distance_kilometers{server_id="55164"} 124.72244390454571
speedtest_server_info{server_country="Germany",server_id="55164",server_lat="49.4521",server_lon="11.0767",server_name="Nurnberg",user_isp="Deutsche Telekom AG"} 1
```
