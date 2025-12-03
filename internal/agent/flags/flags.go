package flags

import (
	"flag"
	"log"
	"os"
	"strconv"
)

const (
	EnvAddress        = "ADDRESS"
	EnvReportInterval = "REPORT_INTERVAL"
	EnvPollInterval   = "POLL_INTERVAL"
	EnvKey            = "KEY"
)

var FlagMetricsServer string
var FlagPollInterval int
var FlagReportPollInterval int
var FlagKey string

func ParseFlags() {
	// Parse metrics server address
	var exist bool
	FlagMetricsServer, exist = os.LookupEnv(EnvAddress)
	if !exist {
		flag.StringVar(&FlagMetricsServer, "a", "localhost:8080", "address and port of metric server")
	}

	// Parse poll interval
	var pollInterval string
	pollInterval, exist = os.LookupEnv(EnvPollInterval)
	if exist {
		var err error
		FlagPollInterval, err = strconv.Atoi(pollInterval)

		if err != nil {
			log.Fatal("unable to parse POLL_INTERVAL env value")
		}
	} else {
		flag.IntVar(&FlagPollInterval, "p", 2, "poll interval in seconds")
	}

	// Parse flagreport interval
	var reportPollInterval string
	reportPollInterval, exist = os.LookupEnv(EnvReportInterval)
	if exist {
		var err error
		FlagReportPollInterval, err = strconv.Atoi(reportPollInterval)

		if err != nil {
			log.Fatal("unable to parse REPORT_INTERVAL env value")
		}
	} else {
		flag.IntVar(&FlagReportPollInterval, "r", 10, "report interval in seconds")
	}

	// Parse key - всегда регистрируем флаг
	flag.StringVar(&FlagKey, "k", "", "hash key for signing requests")

	flag.Parse()

	// Перезаписываем значение из переменной окружения, если она задана
	if key, exist := os.LookupEnv(EnvKey); exist {
		FlagKey = key
	}
}
