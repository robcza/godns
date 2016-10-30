package main

import (
	"os"
	"os/signal"
	"runtime"
	"time"
)

var (
	logger *GoDNSLogger
)

func main() {

	initLogger()

	server := &Server{
		host:     settings.Server.Host,
		port:     settings.Server.Port,
		rTimeout: time.Duration(settings.Server.ReadTimeout) * time.Second,
		wTimeout: time.Duration(settings.Server.WriteTimeout) * time.Second,
	}

	server.Run()

	logger.Info("godns %s start", settings.Version)
	logger.Info("godns %s start", settings.Version)
	logger.Info("Core Backend Settings")
	logger.Info("  FitResponseTime: %d ms", settings.Backend.FitResponseTime)
	logger.Info("  HardRequestTimeout: %d ms", settings.Backend.HardRequestTimeout)
	logger.Info("  SleepWhenDisabled: %d ms", settings.Backend.SleepWhenDisabled)

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

	if settings.Log.Stdout {
		logger.SetLogger("console", nil)
	}

	if settings.Log.File != "" {
		config := map[string]interface{}{"file": settings.Log.File}
		logger.SetLogger("file", config)
	}

	logger.SetLevel(settings.Log.LogLevel())
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
