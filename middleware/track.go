package middleware

import (
	"log/slog"
	"net/http"
)

func MWTrackAddr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.RequestURI, "method", r.Method, "remote_addr", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
