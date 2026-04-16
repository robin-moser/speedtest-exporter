package exporter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var errTooManyRequests = errors.New("another scrape is already in flight")

var (
	descUp = prometheus.NewDesc(
		"speedtest_up",
		"Whether the speedtest succeeded.",
		nil,
		nil,
	)
	descScrapeDuration = prometheus.NewDesc(
		"speedtest_scrape_duration_seconds",
		"Total runtime of the speedtest scrape.",
		nil,
		nil,
	)
	descLatency = prometheus.NewDesc(
		"speedtest_latency_seconds",
		"Measured speedtest latency.",
		nil,
		nil,
	)
	descLatencyMin = prometheus.NewDesc(
		"speedtest_latency_min_seconds",
		"Minimum measured latency during the test.",
		nil,
		nil,
	)
	descLatencyMax = prometheus.NewDesc(
		"speedtest_latency_max_seconds",
		"Maximum measured latency during the test.",
		nil,
		nil,
	)
	descLatencyJitter = prometheus.NewDesc(
		"speedtest_latency_jitter_seconds",
		"Measured jitter (latency variation).",
		nil,
		nil,
	)
	descDownload = prometheus.NewDesc(
		"speedtest_download_bytes_per_second",
		"Measured speedtest download speed in bytes per second.",
		nil,
		nil,
	)
	descUpload = prometheus.NewDesc(
		"speedtest_upload_bytes_per_second",
		"Measured speedtest upload speed in bytes per second.",
		nil,
		nil,
	)
	descServerInfo = prometheus.NewDesc(
		"speedtest_server_info",
		"Static information about the selected speedtest server.",
		[]string{"server_id", "server_name", "server_country", "user_isp", "server_lat", "server_lon"},
		nil,
	)
	descServerDistance = prometheus.NewDesc(
		"speedtest_server_distance_kilometers",
		"Distance to the selected speedtest server in kilometers.",
		[]string{"server_id"},
		nil,
	)
)

type RunFunc func(context.Context, Config) (Result, error)

type Handler struct {
	config Config
	run    RunFunc
	gate   chan struct{}
}

func NewHandler(config Config, run RunFunc) *Handler {
	return &Handler{
		config: config,
		run:    run,
		gate:   make(chan struct{}, 1),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/metrics" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	select {
	case h.gate <- struct{}{}:
		defer func() { <-h.gate }()
	default:
		http.Error(w, errTooManyRequests.Error(), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.config.Timeout)
	defer cancel()

	started := time.Now()
	result, err := h.run(ctx, h.config)
	if err != nil {
		log.Printf("speedtest failed: %v", err)
	}
	if result.ScrapeDurationSeconds == 0 {
		result.ScrapeDurationSeconds = time.Since(started).Seconds()
	}

	registry := prometheus.NewRegistry()
	if err := registry.Register(metricsCollector{result: result}); err != nil {
		http.Error(w, fmt.Sprintf("register metrics: %v", err), http.StatusInternalServerError)
		return
	}
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

type metricsCollector struct {
	result Result
}

func (c metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c metricsCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		descUp,
		prometheus.GaugeValue,
		boolFloat(c.result.Success),
	)
	ch <- prometheus.MustNewConstMetric(
		descScrapeDuration,
		prometheus.GaugeValue,
		c.result.ScrapeDurationSeconds,
	)

	if !c.result.Success {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		descLatency,
		prometheus.GaugeValue,
		c.result.LatencySeconds,
	)
	ch <- prometheus.MustNewConstMetric(
		descLatencyMin,
		prometheus.GaugeValue,
		c.result.MinLatencySeconds,
	)
	ch <- prometheus.MustNewConstMetric(
		descLatencyMax,
		prometheus.GaugeValue,
		c.result.MaxLatencySeconds,
	)
	ch <- prometheus.MustNewConstMetric(
		descLatencyJitter,
		prometheus.GaugeValue,
		c.result.JitterSeconds,
	)
	ch <- prometheus.MustNewConstMetric(
		descDownload,
		prometheus.GaugeValue,
		c.result.DownloadBytesPerSecond,
	)
	ch <- prometheus.MustNewConstMetric(
		descUpload,
		prometheus.GaugeValue,
		c.result.UploadBytesPerSecond,
	)
	ch <- prometheus.MustNewConstMetric(
		descServerInfo,
		prometheus.GaugeValue,
		1,
		fmt.Sprintf("%d", c.result.ServerID),
		c.result.ServerName,
		c.result.ServerCountry,
		c.result.UserISP,
		c.result.ServerLat,
		c.result.ServerLon,
	)
	ch <- prometheus.MustNewConstMetric(
		descServerDistance,
		prometheus.GaugeValue,
		c.result.Distance,
		fmt.Sprintf("%d", c.result.ServerID),
	)
}

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
