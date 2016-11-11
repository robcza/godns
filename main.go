package main

import _ "net/http/pprof"
import (
	"os"
	"os/signal"
	"runtime"
	"time"
	"log"
	"net/http"
)

var (
	logger *GoDNSLogger
)

func main() {
	// Profiler
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	initLogger()

	server := &Server{
		host:     settings.BIND_HOST,
		port:     settings.BIND_PORT,
		rTimeout: time.Duration(settings.GODNS_READ_TIMEOUT) * time.Millisecond,
		wTimeout: time.Duration(settings.GODNS_WRITE_TIMEOUT) * time.Millisecond,
	}

	server.Run()

	logger.Info("godns %s start", settings.Version)
	logger.Info("godns %s start", settings.Version)
	logger.Info("Core Backend Settings")
	logger.Info("  FitResponseTime: %d ms", settings.ORACULUM_API_FIT_TIMEOUT)
	logger.Info("  HardRequestTimeout: %d ms", settings.ORACULUM_API_TIMEOUT)
	logger.Info("  SleepWhenDisabled: %d ms", settings.ORACULUM_SLEEP_WHEN_DISABLED)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	forever:
	for {
		select {
		case <-sig:
			logger.Info("signal received, stopping")
			break forever
		}
	}

}

func initLogger() {
	logger = NewLogger()

	if settings.LOG_STDOUT {
		logger.SetLogger("console", nil)
	}

	if settings.LOG_FILE != "" {
		config := map[string]interface{}{"file": settings.LOG_FILE}
		logger.SetLogger("file", config)
	}

	logger.SetLevel(settings.LogLevel())
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
