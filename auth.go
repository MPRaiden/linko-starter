package main

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	pkgerr "github.com/pkg/errors"
)

type contextKey string

const UserContextKey contextKey = "user"

type LogContext struct {
	Username string
	Error    error
}

const logContextKey contextKey = "log_context"

var allowedUsers = map[string]string{
	"frodo":   "$2a$10$B6O/n6teuCzpuh66jrUAdeaJ3WvXcxRkzpN0x7H.di9G9e/NGb9Me",
	"samwise": "$2a$10$EWZpvYhUJtJcEMmm/IBOsOGIcpxUnGIVMRiDlN/nxl1RRwWGkJtty",
	// frodo: "ofTheNineFingers"
	// samwise: "theStrong"
	"saruman": "invalidFormat",
}

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok {
			httpError(r.Context(), w, http.StatusUnauthorized, pkgerr.WithStack(errors.New("unauthorized")))
			return
		}
		stored, exists := allowedUsers[user]
		if !exists {
			httpError(r.Context(), w, http.StatusUnauthorized, pkgerr.WithStack(errors.New("unauthorized")))
			return
		}
		ok, err := s.validatePassword(password, stored)
		if err != nil {
			s.logger.Error("error validating password", "user", user, "error", err)
			httpError(r.Context(), w, http.StatusInternalServerError, pkgerr.WithStack(errors.New("internal server error")))
			return
		}
		if !ok {
			httpError(r.Context(), w, http.StatusUnauthorized, pkgerr.WithStack(errors.New("unauthorized")))
			return
		}
		ctx := r.Context().Value(logContextKey).(*LogContext)
		ctx.Username = user
		r = r.WithContext(context.WithValue(r.Context(), UserContextKey, user))
		next.ServeHTTP(w, r)
	})
}

func (s *server) validatePassword(password, stored string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, pkgerr.WithStack(err)
	}
	return true, nil
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if logCtx, ok := ctx.Value(logContextKey).(*LogContext); ok {
		logCtx.Error = err
	}
	var errTxt string
	switch status {
	case 401:
		errTxt = http.StatusText(401)
	case 403:
		errTxt = http.StatusText(403)
	case 500:
		errTxt = http.StatusText(500)
	default:
		errTxt = err.Error()
	}

	http.Error(w, errTxt, status)
}
