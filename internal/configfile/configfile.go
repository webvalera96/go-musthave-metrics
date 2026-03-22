package configfile

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const EnvConfig = "CONFIG"

// ResolvePath: флаг -c/-config в командной строке имеет приоритет над переменной CONFIG.
func ResolvePath() string {
	if p := parseFlagFromArgs(); p != "" {
		return p
	}
	return os.Getenv(EnvConfig)
}

func parseFlagFromArgs() string {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "-config" || a == "--config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "-c="):
			return strings.TrimPrefix(a, "-c=")
		case strings.HasPrefix(a, "-config="):
			return strings.TrimPrefix(a, "-config=")
		case strings.HasPrefix(a, "--config="):
			return strings.TrimPrefix(a, "--config=")
		}
	}
	return ""
}

// FilterArgs удаляет из argv пару -c/-config <path> и -c=, -config=, чтобы дальше корректно отработал flag.Parse.
func FilterArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	out = append(out, args[0])
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "-config" || a == "--config":
			if i+1 < len(args) {
				i++
			}
			continue
		case strings.HasPrefix(a, "-c=") || strings.HasPrefix(a, "-config=") || strings.HasPrefix(a, "--config="):
			continue
		default:
			out = append(out, a)
		}
	}
	return out
}

func parseDurationToSeconds(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	sec := int(d / time.Second)
	if sec < 0 {
		return 0, fmt.Errorf("negative duration")
	}
	return sec, nil
}

// Server описывает JSON-файл конфигурации сервера.
type Server struct {
	Address       *string `json:"address,omitempty"`
	Restore       *bool   `json:"restore,omitempty"`
	StoreInterval *string `json:"store_interval,omitempty"`
	StoreFile     *string `json:"store_file,omitempty"`
	DatabaseDSN   *string `json:"database_dsn,omitempty"`
	CryptoKey     *string `json:"crypto_key,omitempty"`
	Key           *string `json:"key,omitempty"`
	AuditFile     *string `json:"audit_file,omitempty"`
	AuditURL      *string `json:"audit_url,omitempty"`
	TrustedSubnet *string `json:"trusted_subnet,omitempty"`
	GRPCAddr      *string `json:"grpc_address,omitempty"`
}

// LoadServer читает JSON (приоритет ниже, чем у флагов и env).
func LoadServer(path string) (*Server, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Server
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ApplyServerIntervalSeconds выставляет store_interval из строки длительности («1s»).
func ApplyServerIntervalSeconds(s *Server) (int, bool, error) {
	if s == nil || s.StoreInterval == nil {
		return 0, false, nil
	}
	sec, err := parseDurationToSeconds(*s.StoreInterval)
	if err != nil {
		return 0, false, fmt.Errorf("store_interval: %w", err)
	}
	return sec, true, nil
}

// Agent описывает JSON-файл конфигурации агента.
type Agent struct {
	Address        *string `json:"address,omitempty"`
	ReportInterval *string `json:"report_interval,omitempty"`
	PollInterval   *string `json:"poll_interval,omitempty"`
	CryptoKey      *string `json:"crypto_key,omitempty"`
	Key            *string `json:"key,omitempty"`
	RateLimit      *int    `json:"rate_limit,omitempty"`
	GRPCAddr       *string `json:"grpc_address,omitempty"`
}

// LoadAgent читает JSON.
func LoadAgent(path string) (*Agent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a Agent
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ApplyAgentIntervalSeconds возвращает poll/report в секундах из полей длительности.
func ApplyAgentIntervalSeconds(a *Agent) (pollSec int, pollOK bool, reportSec int, reportOK bool, err error) {
	if a == nil {
		return 0, false, 0, false, nil
	}
	if a.PollInterval != nil {
		sec, e := parseDurationToSeconds(*a.PollInterval)
		if e != nil {
			return 0, false, 0, false, fmt.Errorf("poll_interval: %w", e)
		}
		pollSec, pollOK = sec, true
	}
	if a.ReportInterval != nil {
		sec, e := parseDurationToSeconds(*a.ReportInterval)
		if e != nil {
			return 0, false, 0, false, fmt.Errorf("report_interval: %w", e)
		}
		reportSec, reportOK = sec, true
	}
	return pollSec, pollOK, reportSec, reportOK, nil
}
