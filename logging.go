package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"boot.dev/linko/internal/linkoerr"
	pkgerr "github.com/pkg/errors"
)

type closeFunc func() error

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

type multiError interface {
	error
	Unwrap() []error
}

func errorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		{
			Key:   "message",
			Value: slog.StringValue(err.Error()),
		},
	}
	attrs = append(attrs, linkoerr.Attrs(err)...)

	if stackErr, ok := errors.AsType[stackTracer](err); ok {
		attrs = append(attrs, slog.Attr{
			Key:   "stack_trace",
			Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
		})
	}

	return attrs
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}
		if multiErr, ok := errors.AsType[multiError](err); ok {
			errorsGroup := []slog.Attr{}
			for i, e := range multiErr.Unwrap() {
				errorsGroup = append(errorsGroup, slog.GroupAttrs(fmt.Sprintf("error_%d", i+1), errorAttrs(e)...))
			}
			return slog.GroupAttrs("errors", errorsGroup...)
		}

		attrs := errorAttrs(err)

		return slog.GroupAttrs("error", attrs...)
	}
	return a
}

func initializeLogger() (*slog.Logger, closeFunc, error) {
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: replaceAttr,
		Level:       slog.LevelDebug,
	})

	fileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		bufferedFile := bufio.NewWriterSize(logFile, 8192)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file: %v", err)
		}
		infoHandler := slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
			ReplaceAttr: replaceAttr,
			Level:       slog.LevelInfo,
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
