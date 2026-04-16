package exporter

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	libspeedtest "github.com/showwin/speedtest-go/speedtest"
)

const AutoSelectServerID = -1

type Config struct {
	Port           string
	PingMode       libspeedtest.Proto
	ServerID       int
	ServerFallback bool
	Timeout        time.Duration
}

func LoadConfig() (Config, error) {
	config := Config{
		Port:           "9090",
		PingMode:       libspeedtest.TCP,
		ServerID:       AutoSelectServerID,
		ServerFallback: false,
		Timeout:        60 * time.Second,
	}

	if value := os.Getenv("PORT"); value != "" {
		config.Port = value
	}

	if value := os.Getenv("PING_MODE"); value != "" {
		pingMode, err := parsePingMode(value)
		if err != nil {
			return Config{}, err
		}
		config.PingMode = pingMode
	}

	if value := os.Getenv("SERVER_ID"); value != "" {
		serverID, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse SERVER_ID: %w", err)
		}
		config.ServerID = serverID
	}

	if value := os.Getenv("SERVER_FALLBACK"); value != "" {
		fallback, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse SERVER_FALLBACK: %w", err)
		}
		config.ServerFallback = fallback
	}

	if value := os.Getenv("TIMEOUT"); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse TIMEOUT: %w", err)
		}
		if seconds <= 0 {
			return Config{}, fmt.Errorf("parse TIMEOUT: must be greater than 0")
		}
		config.Timeout = time.Duration(seconds) * time.Second
	}

	if config.Port == "" {
		return Config{}, fmt.Errorf("PORT must not be empty")
	}

	return config, nil
}

func parsePingMode(value string) (libspeedtest.Proto, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "http":
		return libspeedtest.HTTP, nil
	case "tcp":
		return libspeedtest.TCP, nil
	case "icmp":
		return libspeedtest.ICMP, nil
	default:
		return 0, fmt.Errorf("parse PING_MODE: must be one of http, tcp, icmp")
	}
}
