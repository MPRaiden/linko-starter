package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"

	"boot.dev/linko/internal/linkoerr"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	pkgerr "github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
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
	var sensitiveKeys = []string{"password", "key", "apikey", "secret", "pin", "creditcardno", "user"}

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
	if slices.Contains(sensitiveKeys, a.Key) {
		return slog.String(a.Key, "[REDACTED]")
	}
	return a
}

func initializeLogger() (*slog.Logger, closeFunc, error) {
	w := os.Stderr
	// if in terminal env we want color in logs, otherwise no
	isTermEnv := isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd())
	debugHandler := tint.NewTextHandler(os.Stderr, &tint.Options{
		ReplaceAttr: replaceAttr,
		Level:       slog.LevelDebug,
		NoColor:     !isTermEnv,
	})

	fileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		fileLogger := &lumberjack.Logger{
			Filename:   fileName,
			MaxSize:    1,
			MaxAge:     28,
			MaxBackups: 10,
			LocalTime:  false,
			Compress:   true,
		}

		infoHandler := slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{
			ReplaceAttr: replaceAttr,
			Level:       slog.LevelInfo,
		})

		logger := slog.New(slog.NewMultiHandler(
			debugHandler,
			infoHandler,
		))

		closeLogger := func() error {
			err := fileLogger.Close()
			if err != nil {
				return err
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

func redactIP(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return address
	}

	split := strings.Split(ip.String(), ".")
	split[len(split)-1] = "x"
	redacted := strings.Join(split, ".")

	return redacted
}
