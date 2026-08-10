package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger() (*slog.Logger, closeFunc, error) {
	fileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		bufferedFile := bufio.NewWriterSize(logFile, 8192)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file: %v", err)
		}
		multiWritter := io.MultiWriter(os.Stderr, bufferedFile)
		logger := slog.New(slog.NewTextHandler(multiWritter, nil))

		closeLogger := func() error {
			err := bufferedFile.Flush()
			if err != nil {
				return fmt.Errorf("failed to flush buffered file: %w", err)
			}
			err = logFile.Close()
			if err != nil {
				return fmt.Errorf("failed to close logFile: %w", err)
			}
			return nil
		}
		return logger, closeLogger, nil
	}

	singleWritter := os.Stderr
	logger := slog.New(slog.NewTextHandler(singleWritter, nil))

	closeFuncNoOp := func() error {
		return nil
	}

	return logger, closeFuncNoOp, nil
}
