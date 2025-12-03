package flags

import (
	"flag"
	"log"
	"os"
	"strconv"
)

const (
	EnvAddress         = "ADDRESS"
	EnvStoreInterval   = "STORE_INTERVAL"
	EnvFileStoragePath = "FILE_STORAGE_PATH"
	EnvRestore         = "RESTORE"
	EnvDatabaseDSN     = "DATABASE_DSN"
	EnvKey             = "KEY"
)

var FlagRunAddr string
var FlagStoreInterval int
var FlagStoragePath string
var FlagRestore bool
var FlagDatabaseDSN string
var FlagKey string

func ParseFlags() {
	var exist bool

	var runAddr string
	runAddr, exist = os.LookupEnv(EnvAddress)

	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")

	if exist {
		FlagRunAddr = runAddr
	}

	var storeInterval string
	storeInterval, exist = os.LookupEnv(EnvStoreInterval)

	flag.IntVar(&FlagStoreInterval, "i", 300, "time interval to save data on disk")

	if exist {
		var err error
		FlagStoreInterval, err = strconv.Atoi(storeInterval)
		if err != nil {
			log.Fatalf("unable to parse %s env value", EnvStoreInterval)
		}
	}

	var fileStoragePath string
	fileStoragePath, exist = os.LookupEnv(EnvFileStoragePath)

	flag.StringVar(&FlagStoragePath, "f", "", "path to saved data in file")

	if exist {
		FlagStoragePath = fileStoragePath
	}

	var restore string
	restore, exist = os.LookupEnv(EnvRestore)

	flag.BoolVar(&FlagRestore, "r", false, "restore from file or database")

	if exist {
		var err error
		FlagRestore, err = strconv.ParseBool(restore)
		if err != nil {
			log.Fatalf("unabble to parse %s env value", EnvRestore)
		}
	}

	var databaseDSN string
	databaseDSN, exist = os.LookupEnv(EnvDatabaseDSN)

	flag.StringVar(&FlagDatabaseDSN, "d", "", "database DSN")

	if exist {
		FlagDatabaseDSN = databaseDSN
	}

	// Parse key - всегда регистрируем флаг
	flag.StringVar(&FlagKey, "k", "", "hash key for signing requests and responses")

	flag.Parse()

	// Перезаписываем значение из переменной окружения, если она задана
	if key, exist := os.LookupEnv(EnvKey); exist {
		FlagKey = key
	}

}
