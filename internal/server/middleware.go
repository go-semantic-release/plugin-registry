package server

import (
	"fmt"
	"net/http"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.AdminAccessToken == "" {
			s.writeJSONError(w, r, http.StatusUnauthorized, fmt.Errorf("no access token configured"))
			return
		}
		if r.Header.Get("Authorization") != s.config.AdminAccessToken {
			s.requestLogger(r).Warnf("invalid access token from %s", r.RemoteAddr)
			s.writeJSONError(w, r, http.StatusUnauthorized, fmt.Errorf("invalid access token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requestLogger(r).Infof("%s %s (%s)", r.Method, r.URL.EscapedPath(), r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.writeJSONError(w, r, http.StatusInternalServerError, fmt.Errorf("panic: %v", rec))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
