package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.adminAccessToken == "" {
			s.writeJSONError(w, r, http.StatusUnauthorized, fmt.Errorf("no access token configured"))
			return
		}
		if r.Header.Get("Authorization") != s.adminAccessToken {
			s.writeJSONError(w, r, http.StatusUnauthorized, fmt.Errorf("invalid access token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Printf("[%s] %s %s (%s)", middleware.GetReqID(r.Context()), r.Method, r.URL.EscapedPath(), r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
