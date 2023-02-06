package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

func (s *Server) setContentTypeJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

func (s *Server) writeJSON(w http.ResponseWriter, d any) {
	s.setContentTypeJSON(w)
	err := json.NewEncoder(w).Encode(d)
	if err != nil {
		s.log.Error(err)
	}
}

func (s *Server) writeJSONError(w http.ResponseWriter, r *http.Request, statusCode int, err error, alternativeMessage ...string) {
	errMsg := err.Error()
	s.log.WithFields(logrus.Fields{
		LogFieldRequestID: middleware.GetReqID(r.Context()),
		LogFieldHTTPRequest: map[string]any{
			"requestMethod": r.Method,
			"requestUrl":    r.URL.EscapedPath(),
			"status":        statusCode,
		},
	}).Errorf("error: %s", errMsg)

	s.setContentTypeJSON(w)
	w.WriteHeader(statusCode)

	if len(alternativeMessage) > 0 {
		errMsg = strings.Join(alternativeMessage, " ")
	}
	s.writeJSON(w, map[string]string{"error": errMsg})
}

func (s *Server) requestLogger(r *http.Request) *logrus.Entry {
	return s.log.WithFields(logrus.Fields{
		LogFieldRequestID: middleware.GetReqID(r.Context()),
		LogFieldHTTPRequest: map[string]any{
			"requestMethod": r.Method,
			"requestUrl":    r.URL.EscapedPath(),
		},
	})
}
