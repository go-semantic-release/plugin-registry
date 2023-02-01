package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.AdminAccessToken == "" {
			s.writeJSONError(w, r, http.StatusUnauthorized, fmt.Errorf("no access token configured"))
			return
		}
		if r.Header.Get("Authorization") != s.config.AdminAccessToken {
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

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Errorf("panic: %v", rec)
				s.writeJSONError(w, r, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if k, ok := s.getFromCache(s.getCacheKeyFromRequest(r)); ok {
			w.Header().Set("X-Cache", "HIT")
			s.writeJSON(w, k)
			return
		}
		next.ServeHTTP(w, r)
	})
}
