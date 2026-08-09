package main

import (
	"io"
	"log"
	"os"
)

func initializeLogger() *log.Logger {
	fileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		multiWritter := io.MultiWriter(os.Stderr, logFile)
		logger := log.New(multiWritter, "", log.LstdFlags)

		return logger
	}

	singleWritter := os.Stderr
	logger := log.New(singleWritter, "", log.LstdFlags)

	return logger
}
