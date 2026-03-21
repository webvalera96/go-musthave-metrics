package flags

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/webvalera96/go-musthave-metrics/internal/configfile"
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

// NewServerConfig парсит JSON (низший приоритет), затем флаги, затем env (высший приоритет).
func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{
		RunAddr:       ":8080",
		StoreInterval: 300,
		StoragePath:   "",
		Restore:       false,
		DatabaseDSN:   "",
		Key:           "",
		CryptoKey:     "",
		AuditFile:     "",
		AuditURL:      "",
	}

	if p := configfile.ResolvePath(); p != "" {
		s, err := configfile.LoadServer(p)
		if err != nil {
			log.Fatalf("server config file: %v", err)
		}
		if err := applyServerFile(cfg, s); err != nil {
			log.Fatal(err)
		}
	}

	flag.StringVar(&cfg.RunAddr, "a", cfg.RunAddr, "address and port to run server")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "time interval to save data on disk")
	flag.StringVar(&cfg.StoragePath, "f", cfg.StoragePath, "path to saved data in file")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore from file or database")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database DSN")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "hash key for signing requests and responses")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to PEM file with RSA private key for decrypting agent requests")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "path to file for audit logs")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "full URL to send audit logs via POST")

	_ = flag.CommandLine.Parse(configfile.FilterArgs(os.Args)[1:])

	runAddr, runAddrEnv := os.LookupEnv(EnvAddress)
	storeIntervalStr, storeIntervalEnv := os.LookupEnv(EnvStoreInterval)
	fileStoragePath, fileStoragePathEnv := os.LookupEnv(EnvFileStoragePath)
	restoreStr, restoreEnv := os.LookupEnv(EnvRestore)
	databaseDSN, databaseDSNEnv := os.LookupEnv(EnvDatabaseDSN)

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

func applyServerFile(cfg *ServerConfig, s *configfile.Server) error {
	if s == nil {
		return nil
	}
	if s.Address != nil {
		cfg.RunAddr = *s.Address
	}
	if s.Restore != nil {
		cfg.Restore = *s.Restore
	}
	sec, ok, err := configfile.ApplyServerIntervalSeconds(s)
	if err != nil {
		return err
	}
	if ok {
		cfg.StoreInterval = sec
	}
	if s.StoreFile != nil {
		cfg.StoragePath = *s.StoreFile
	}
	if s.DatabaseDSN != nil {
		cfg.DatabaseDSN = *s.DatabaseDSN
	}
	if s.CryptoKey != nil {
		cfg.CryptoKey = *s.CryptoKey
	}
	if s.Key != nil {
		cfg.Key = *s.Key
	}
	if s.AuditFile != nil {
		cfg.AuditFile = *s.AuditFile
	}
	if s.AuditURL != nil {
		cfg.AuditURL = *s.AuditURL
	}
	return nil
}
