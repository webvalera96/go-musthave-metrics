package flags

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/webvalera96/go-musthave-metrics/internal/configfile"
)

// AgentConfig — конфигурация агента (флаги и переменные окружения), передаётся через DI.
type AgentConfig struct {
	MetricsServer      string
	PollInterval       int
	ReportPollInterval int
	Key                string
	RateLimit          int
	CryptoKey          string
	GRPCAddr           string
}

// NewAgentConfig парсит JSON (низший приоритет), затем флаги, затем env (высший приоритет).
func NewAgentConfig() *AgentConfig {
	cfg := &AgentConfig{
		MetricsServer:      "localhost:8080",
		PollInterval:       2,
		ReportPollInterval: 10,
		Key:                "",
		RateLimit:          1,
		CryptoKey:          "",
		GRPCAddr:           "",
	}

	if p := configfile.ResolvePath(); p != "" {
		a, err := configfile.LoadAgent(p)
		if err != nil {
			log.Fatalf("agent config file: %v", err)
		}
		if err := applyAgentFile(cfg, a); err != nil {
			log.Fatal(err)
		}
	}

	flag.StringVar(&cfg.MetricsServer, "a", cfg.MetricsServer, "address and port of metric server")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "poll interval in seconds")
	flag.IntVar(&cfg.ReportPollInterval, "r", cfg.ReportPollInterval, "report interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "hash key for signing requests")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to PEM file with RSA public key for request encryption")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "rate limit for concurrent requests")
	flag.StringVar(&cfg.GRPCAddr, "grpc", cfg.GRPCAddr, "gRPC address of Metrics server (optional; if set, metrics are sent via gRPC)")

	_ = flag.CommandLine.Parse(configfile.FilterArgs(os.Args)[1:])

	if v, ok := os.LookupEnv(EnvAddress); ok {
		cfg.MetricsServer = v
	}
	if v, ok := os.LookupEnv(EnvPollInterval); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatal("unable to parse POLL_INTERVAL env value")
		}
		cfg.PollInterval = n
	}
	if v, ok := os.LookupEnv(EnvReportInterval); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatal("unable to parse REPORT_INTERVAL env value")
		}
		cfg.ReportPollInterval = n
	}
	if v, ok := os.LookupEnv(EnvKey); ok {
		cfg.Key = v
	}
	if v, ok := os.LookupEnv(EnvRateLimit); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatal("unable to parse RATE_LIMIT env value")
		}
		cfg.RateLimit = n
	}
	if v, ok := os.LookupEnv(EnvCryptoKey); ok {
		cfg.CryptoKey = v
	}
	if v, ok := os.LookupEnv(EnvGRPCAddress); ok {
		cfg.GRPCAddr = v
	}

	return cfg
}

func applyAgentFile(cfg *AgentConfig, a *configfile.Agent) error {
	if a == nil {
		return nil
	}
	if a.Address != nil {
		cfg.MetricsServer = *a.Address
	}
	pollSec, pollOK, reportSec, reportOK, err := configfile.ApplyAgentIntervalSeconds(a)
	if err != nil {
		return err
	}
	if pollOK {
		cfg.PollInterval = pollSec
	}
	if reportOK {
		cfg.ReportPollInterval = reportSec
	}
	if a.Key != nil {
		cfg.Key = *a.Key
	}
	if a.RateLimit != nil {
		cfg.RateLimit = *a.RateLimit
	}
	if a.CryptoKey != nil {
		cfg.CryptoKey = *a.CryptoKey
	}
	if a.GRPCAddr != nil {
		cfg.GRPCAddr = *a.GRPCAddr
	}
	return nil
}
