package exporter

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	libspeedtest "github.com/showwin/speedtest-go/speedtest"
)

type Result struct {
	Success                bool
	ScrapeDurationSeconds  float64
	LatencySeconds         float64
	MinLatencySeconds      float64
	MaxLatencySeconds      float64
	JitterSeconds          float64
	DownloadBytesPerSecond float64
	UploadBytesPerSecond   float64
	UserISP                string
	ServerID               int
	ServerName             string
	ServerCountry          string
	ServerLat              string
	ServerLon              string
	Distance               float64
}

func RunSpeedtest(ctx context.Context, config Config) (Result, error) {
	started := time.Now()
	var result Result
	fail := func(format string, args ...any) (Result, error) {
		result.ScrapeDurationSeconds = time.Since(started).Seconds()
		return result, fmt.Errorf(format, args...)
	}

	client := libspeedtest.New(libspeedtest.WithUserConfig(&libspeedtest.UserConfig{PingMode: config.PingMode}))
	user, err := client.FetchUserInfoContext(ctx)
	if err != nil {
		return fail("fetch user info: %w", err)
	}
	result.UserISP = user.Isp

	servers, err := client.FetchServerListContext(ctx)
	if err != nil {
		return fail("fetch server list: %w", err)
	}

	server, err := selectServer(servers, config)
	if err != nil {
		return fail("%w", err)
	}

	serverID, err := strconv.Atoi(server.ID)
	if err != nil {
		return fail("parse server id %q: %w", server.ID, err)
	}
	result.ServerID = serverID
	result.ServerName = server.Name
	result.ServerCountry = server.Country
	result.ServerLat = server.Lat
	result.ServerLon = server.Lon
	result.Distance = server.Distance

	if err := server.PingTestContext(ctx, nil); err != nil {
		return fail("ping test: %w", err)
	}
	result.LatencySeconds = server.Latency.Seconds()
	result.MinLatencySeconds = server.MinLatency.Seconds()
	result.MaxLatencySeconds = server.MaxLatency.Seconds()
	result.JitterSeconds = server.Jitter.Seconds()

	if err := server.DownloadTestContext(ctx); err != nil {
		return fail("download test: %w", err)
	}
	result.DownloadBytesPerSecond = float64(server.DLSpeed)

	if err := server.UploadTestContext(ctx); err != nil {
		return fail("upload test: %w", err)
	}
	result.UploadBytesPerSecond = float64(server.ULSpeed)

	result.Success = true
	result.ScrapeDurationSeconds = time.Since(started).Seconds()
	return result, nil
}

func selectServer(servers libspeedtest.Servers, config Config) (*libspeedtest.Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("no speedtest servers available")
	}

	if config.ServerID == AutoSelectServerID {
		selected, err := servers.FindServer(nil)
		if err != nil {
			return nil, fmt.Errorf("select default server: %w", err)
		}
		if len(selected) == 0 || selected[0] == nil {
			return nil, errors.New("select default server: no matching server")
		}
		return selected[0], nil
	}

	for _, server := range servers {
		if server != nil && server.ID == strconv.Itoa(config.ServerID) {
			return server, nil
		}
	}
	if config.ServerFallback {
		selected, err := servers.FindServer(nil)
		if err != nil {
			return nil, fmt.Errorf("select fallback server: %w", err)
		}
		if len(selected) == 0 || selected[0] == nil {
			return nil, errors.New("select fallback server: no matching server")
		}
		return selected[0], nil
	}
	return nil, fmt.Errorf("find server %d: no matching server", config.ServerID)
}
