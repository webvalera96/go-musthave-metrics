package flags

import (
	"flag"
	"os"
)

const (
	EnvAddress = "ADDRESS"
)

var FlagRunAddr string

func ParseFlags() {
	var exist bool
	FlagRunAddr, exist = os.LookupEnv(EnvAddress)
	if !exist {
		flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
		flag.Parse()
	}
}
