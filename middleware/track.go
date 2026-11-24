package middleware

import (
	"log/slog"
	"net/http"
)

func MWTrackAddr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.RequestURI, "method", r.Method, "remote_addr", r.RemoteAddr)
		slog.Debug("checking XFF", "XFF", r.Header.Get("X-Forward-For"))
		next.ServeHTTP(w, r)
	})
}
