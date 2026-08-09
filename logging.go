package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

func initializeLogger() (*log.Logger, error) {
	fileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		bufferedFile:= bufio.NewWriterSize(logFile, 8192)

		if err != nil {
			return nil, fmt.Errorf("failed to open file: %v", err)
		}
		multiWritter := io.MultiWriter(os.Stderr, bufferedFile)
		logger := log.New(multiWritter, "", log.LstdFlags)

		return logger, nil
	}

	singleWritter := os.Stderr
	logger := log.New(singleWritter, "", log.LstdFlags)

	return logger, nil
}
