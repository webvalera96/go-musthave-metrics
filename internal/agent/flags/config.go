package flags

import (
	"flag"
	"log"
	"os"
	"strconv"
)

// AgentConfig — конфигурация агента (флаги и переменные окружения), передаётся через DI.
type AgentConfig struct {
	MetricsServer       string
	PollInterval        int
	ReportPollInterval  int
	Key                 string
	RateLimit           int
	CryptoKey           string
}

// NewAgentConfig парсит флаги и env и возвращает конфиг. Вызывать до flag.Parse() не нужно — парсинг внутри.
func NewAgentConfig() *AgentConfig {
	var exist bool
	cfg := &AgentConfig{}

	cfg.MetricsServer, exist = os.LookupEnv(EnvAddress)
	if !exist {
		flag.StringVar(&cfg.MetricsServer, "a", "localhost:8080", "address and port of metric server")
	}

	var pollInterval string
	pollInterval, exist = os.LookupEnv(EnvPollInterval)
	if exist {
		var err error
		cfg.PollInterval, err = strconv.Atoi(pollInterval)
		if err != nil {
			log.Fatal("unable to parse POLL_INTERVAL env value")
		}
	} else {
		flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval in seconds")
	}

	var reportPollInterval string
	reportPollInterval, exist = os.LookupEnv(EnvReportInterval)
	if exist {
		var err error
		cfg.ReportPollInterval, err = strconv.Atoi(reportPollInterval)
		if err != nil {
			log.Fatal("unable to parse REPORT_INTERVAL env value")
		}
	} else {
		flag.IntVar(&cfg.ReportPollInterval, "r", 10, "report interval in seconds")
	}

	flag.StringVar(&cfg.Key, "k", "", "hash key for signing requests")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to PEM file with RSA public key for request encryption")
	flag.IntVar(&cfg.RateLimit, "l", 1, "rate limit for concurrent requests")
	flag.Parse()

	if key, exist := os.LookupEnv(EnvKey); exist {
		cfg.Key = key
	}
	if rateLimit, exist := os.LookupEnv(EnvRateLimit); exist {
		var err error
		cfg.RateLimit, err = strconv.Atoi(rateLimit)
		if err != nil {
			log.Fatal("unable to parse RATE_LIMIT env value")
		}
	}
	if v, exist := os.LookupEnv(EnvCryptoKey); exist {
		cfg.CryptoKey = v
	}

	return cfg
}
