package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger() (*slog.Logger, closeFunc, error) {
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	fileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		bufferedFile := bufio.NewWriterSize(logFile, 8192)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file: %v", err)
		}
		infoHandler := slog.NewTextHandler(bufferedFile, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})

		logger := slog.New(slog.NewMultiHandler(
			debugHandler,
			infoHandler,
		))

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

	logger := slog.New(slog.NewMultiHandler(
		debugHandler,
	))

	closeFuncNoOp := func() error {
		return nil
	}

	return logger, closeFuncNoOp, nil
}
