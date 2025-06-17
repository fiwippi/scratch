package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func HttpLogger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			attrs := []any{
				slog.Attr{Key: "url", Value: slog.StringValue(fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI))},
				slog.Attr{Key: "method", Value: slog.StringValue(r.Method)},
				slog.Attr{Key: "ip", Value: slog.StringValue(r.RemoteAddr)},
				slog.Attr{Key: "elapsed", Value: slog.DurationValue(time.Since(start).Round(time.Millisecond))},
			}
			if rw.err != nil {
				attrs = append(attrs, slog.Attr{Key: "error", Value: slog.StringValue(rw.err.Error())})
			}
			slog.With(attrs...).Log(context.Background(), statusLevel(rw.Status()), fmt.Sprintf("%d", rw.Status()))
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	err    error
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	if rw.err != nil {
		rw.ResponseWriter.Write([]byte(rw.err.Error()))
	}
}

func (rw *responseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

func (rw *responseWriter) SetError(err error) {
	rw.err = err
}

func statusLevel(status int) slog.Level {
	switch {
	case status <= 0:
		return slog.LevelWarn
	case status < 400: // For codes in 100s, 200s, 300s
		return slog.LevelInfo
	case status >= 400 && status <= 500:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Error(w http.ResponseWriter, err error) {
	// If we haven't set the logging middleware
	// we won't be able to store any API errors
	if rw, ok := w.(*responseWriter); ok {
		rw.err = err
	}
}
