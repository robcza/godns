package main

import (
	"github.com/kelseyhightower/envconfig"
	"log"
	"runtime"
)

var (
	settings Settings
)

var LogLevelMap = map[string]int{
	"DEBUG":    LevelDebug,
	"INFO":        LevelInfo,
	"NOTICE":    LevelNotice,
	"WARN":        LevelWarn,
	"ERROR":    LevelError,
}

type Settings struct {
	BIND_HOST            string
	BIND_PORT            int
	GODNS_READ_TIMEOUT    int
	GODNS_WRITE_TIMEOUT    int
	RESOLV_CONF_FILE            string
	BACKEND_RESOLVER_RW_TIMEOUT    int
	BACKEND_RESOLVER_TICK        int
	LOG_STDOUT    bool
	LOG_FILE    string
	LOG_LEVEL    string
	ORACULUM_CACHE_BACKEND    string
	ORACULUM_CACHE_EXPIRE    int
	ORACULUM_CACHE_MAXCOUNT    int
	BACKEND_RESOLVERS                []string
	BACKEND_RESOLVERS_EXCLUSIVELY    bool
	ORACULUM_API_FIT_TIMEOUT        int64
	ORACULUM_SLEEP_WHEN_DISABLED    int64
	ORACULUM_API_TIMEOUT            int64
	SINKHOLE_ADDRESS                string
	ORACULUM_ACCESS_TOKEN_VALUE        string
	ORACULUM_ACCESS_TOKEN_KEY        string
	ORACULUM_DISABLED                bool
	ORACULUM_URL                    string
	ORACULUM_IP_ADDRESSES_ENABLED    bool
	Version                string
	NUM_OF_CPUS            int
}

func (s Settings) LogLevel() int {
	l, ok := LogLevelMap[s.LOG_LEVEL]
	if !ok {
		panic("Config error: invalid log level: " + s.LOG_LEVEL)
	}
	return l
}

func init() {
	err := envconfig.Process("SINKIT", &settings)
	if err != nil {
		log.Fatal(err.Error())
	}
	settings.Version = "0.5.0"

	if (settings.NUM_OF_CPUS == 0 || settings.NUM_OF_CPUS > runtime.NumCPU()) {
		settings.NUM_OF_CPUS = runtime.NumCPU()
	}
}
