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
)

var FlagRunAddr string
var FlagStoreInterval int
var FlagStoragePath string
var FlagRestore bool
var FlagDatabaseDSN string

func ParseFlags() {
	var exist bool

	var runAddr string
	runAddr, exist = os.LookupEnv(EnvAddress)
	if exist {
		FlagRunAddr = runAddr
	} else {
		flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	}

	var storeInterval string
	storeInterval, exist = os.LookupEnv(EnvStoreInterval)
	if exist {
		var err error
		FlagStoreInterval, err = strconv.Atoi(storeInterval)
		if err != nil {
			log.Fatalf("unable to parse %s env value", EnvStoreInterval)
		}
	} else {
		flag.IntVar(&FlagStoreInterval, "i", 300, "time interval to save data on disk")
	}

	var fileStoragePath string
	fileStoragePath, exist = os.LookupEnv(EnvFileStoragePath)

	if exist {
		FlagStoragePath = fileStoragePath
	} else {
		flag.StringVar(&FlagStoragePath, "f", "", "path to saved data in file")
	}

	var restore string
	restore, exist = os.LookupEnv(EnvRestore)

	if exist {
		var err error
		FlagRestore, err = strconv.ParseBool(restore)
		if err != nil {
			log.Fatalf("unabble to parse %s env value", EnvRestore)
		}
	} else {
		flag.BoolVar(&FlagRestore, "r", false, "restore from file")
	}

	var databaseDSN string
	databaseDSN, exist = os.LookupEnv(EnvDatabaseDSN)
	if exist {
		FlagDatabaseDSN = databaseDSN
	} else {
		flag.StringVar(&FlagDatabaseDSN, "d", "", "database DSN")
	}

	flag.Parse()

}
