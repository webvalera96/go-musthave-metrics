package flags

import (
	"flag"
	"log"
	"os"
	"strconv"
)

// ServerConfig — конфигурация сервера (флаги и переменные окружения), передаётся через DI.
type ServerConfig struct {
	RunAddr       string
	StoreInterval int
	StoragePath   string
	Restore       bool
	DatabaseDSN   string
	Key           string
	CryptoKey     string
	AuditFile     string
	AuditURL      string
}

// NewServerConfig парсит флаги и env и возвращает конфиг.
func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{}

	runAddr, runAddrEnv := os.LookupEnv(EnvAddress)
	storeIntervalStr, storeIntervalEnv := os.LookupEnv(EnvStoreInterval)
	fileStoragePath, fileStoragePathEnv := os.LookupEnv(EnvFileStoragePath)
	restoreStr, restoreEnv := os.LookupEnv(EnvRestore)
	databaseDSN, databaseDSNEnv := os.LookupEnv(EnvDatabaseDSN)

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "address and port to run server")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "time interval to save data on disk")
	flag.StringVar(&cfg.StoragePath, "f", "", "path to saved data in file")
	flag.BoolVar(&cfg.Restore, "r", false, "restore from file or database")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN")
	flag.StringVar(&cfg.Key, "k", "", "hash key for signing requests and responses")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to PEM file with RSA private key for decrypting agent requests")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "path to file for audit logs")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "full URL to send audit logs via POST")
	flag.Parse()

	if runAddrEnv {
		cfg.RunAddr = runAddr
	}
	if storeIntervalEnv {
		var err error
		cfg.StoreInterval, err = strconv.Atoi(storeIntervalStr)
		if err != nil {
			log.Fatalf("unable to parse %s env value", EnvStoreInterval)
		}
	}
	if fileStoragePathEnv {
		cfg.StoragePath = fileStoragePath
	}
	if restoreEnv {
		var err error
		cfg.Restore, err = strconv.ParseBool(restoreStr)
		if err != nil {
			log.Fatalf("unable to parse %s env value", EnvRestore)
		}
	}
	if databaseDSNEnv {
		cfg.DatabaseDSN = databaseDSN
	}
	if v, exist := os.LookupEnv(EnvKey); exist {
		cfg.Key = v
	}
	if v, exist := os.LookupEnv(EnvCryptoKey); exist {
		cfg.CryptoKey = v
	}
	if v, exist := os.LookupEnv(EnvAuditFile); exist {
		cfg.AuditFile = v
	}
	if v, exist := os.LookupEnv(EnvAuditURL); exist {
		cfg.AuditURL = v
	}

	return cfg
}
