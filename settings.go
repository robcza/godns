package main

import (
	"github.com/kelseyhightower/envconfig"
	"time"
	"log"
	"runtime"
)

var (
	settings Settings
)

var LogLevelMap = map[string]int{
	"DEBUG":  LevelDebug,
	"INFO":   LevelInfo,
	"NOTICE": LevelNotice,
	"WARN":   LevelWarn,
	"ERROR":  LevelError,
}

type Settings struct {
	//Debug = true
	//Debug        bool
	Server       DNSServerSettings
	ResolvConfig ResolvSettings
	Log          LogSettings
	Cache        CacheSettings
	Backend      BackendSettings
	Version      string
	NumCPUs      int
}

type ResolvSettings struct {
	//resolv-file = "/etc/resolv.conf"
	ResolvFile string
	//timeout = 5 # 5 seconds
	//resolver timout 	return time.Duration(r.config.Timeout) * time.Second
	Timeout    int
	//ticker := time.NewTicker(time.Duration(settings.ResolvConfig.Interval) * time.Millisecond)
	//interval = 20 # 20 milliseconds - it's our local resolver
	Interval   int
}
/*
		rTimeout: time.Duration(settings.Server.ReadTimeout) * time.Second,
		wTimeout: time.Duration(settings.Server.WriteTimeout) * time.Second,
 */
type DNSServerSettings struct {
	//host = "127.0.0.1"
	Host string
	//port = 5551
	Port int
	ReadTimeout int
	WriteTimeout int
}

type LogSettings struct {
	Stdout bool   //true
	File   string // "./godns.log"
	Level  string //"DEBUG"  #DEBUG | INFO |NOTICE | WARN | ERROR
}

func (ls LogSettings) LogLevel() int {
	l, ok := LogLevelMap[ls.Level]
	if !ok {
		panic("Config error: invalid log level: " + ls.Level)
	}
	return l
}

//cacheConfig.Backend memory
//			Expire:   time.Duration(cacheConfig.Expire) * time.Second,

type CacheSettings struct {
	//backend = "memory"
	Backend  string
	//expire = 5 #s
	Expire   int
	//maxcount = 0 #If set zero. The Sum of cache itmes will be unlimit.
	Maxcount int
}

/*	logger.Info("Core Backend Settings")
	logger.Info("  FitResponseTime: %d ms", settings.Backend.FitResponseTime)
	logger.Info("  HardRequestTimeout: %d ms", settings.Backend.HardRequestTimeout)
	logger.Info("  SleepWhenDisabled: %d ms", settings.Backend.SleepWhenDisabled)
*/
type BackendSettings struct {
	//backend-recursive-resolvers = [ "8.8.8.8:53" ]
	BackendResolvers []string
	//use-exclusively = true
	UseExclusively bool
	//time.Duration(settings.Backend.FitResponseTime)*time.Millisecond
	//fit-response-time = 200 #ms
	FitResponseTime int64
	//sleep-when-disabled = 10000 #ms
	SleepWhenDisabled int64
	//	return net.DialTimeout(network, addr, time.Duration(settings.Backend.HardRequestTimeout) * time.Millisecond)
	//hard-request-timeout = 200 #ms
	HardRequestTimeout int64
	//os.Getenv("SINKIT_SINKHOLE_IP")
	SinkholeAddress string
	//os.Getenv("SINKIT_ACCESS_TOKEN")
	AccessToken string
	//strconv.ParseBool(os.Getenv("SINKIT_RESOLVER_DISABLE_INFINISPAN"))
	OraculumDisabled bool
	URL string
}
//URL:
//var coreApiServer = "http://"+os.Getenv("SINKIT_CORE_SERVER")+":"+os.Getenv("SINKIT_CORE_SERVER_PORT")+"/sinkit/rest/blacklist/dns"


type Specification struct {
	Debug   bool
	Port    int
	User    string
	Users   []string
	Rate    float32
	Timeout time.Duration
}

func init() {
	var s Specification
	err := envconfig.Process("myapp", &s)
	if err != nil {
		log.Fatal(err.Error())
	}
	settings.Version = "0.5.0"
	settings.NumCPUs = runtime.NumCPU()
}
