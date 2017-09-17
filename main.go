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
	auditor *GoDNSLogger
)

func main() {
	// General system messages, errors...
	initLogger()
	// JSON auditing block events {"client_ip": "<IP>", "domain": "<domain>", "action": "<audit/block>"}
	initAuditor()

	// Profiler
	if (settings.LogLevel() == LevelDebug) {
		go func() {
			log.Println(http.ListenAndServe(settings.BIND_HOST + ":6060", nil))
		}()
	}

	server := &DNSServer{
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
		logger.SetLogger("console", nil, true)
	}

	if settings.LOG_FILE != "" {
		config := map[string]interface{}{"file": settings.LOG_FILE}
		logger.SetLogger("file", config, true)
	}

	logger.SetLevel(settings.LogLevel())
}


func initAuditor() {
	auditor = NewLogger()

	if settings.AUDIT_FILE != "" {
		config := map[string]interface{}{"file": settings.AUDIT_FILE}
		auditor.SetLogger("file", config, false)
	}

	auditor.SetLevel(settings.AuditLevel())
}

func init() {
	runtime.GOMAXPROCS(settings.NUM_OF_CPUS)
}
