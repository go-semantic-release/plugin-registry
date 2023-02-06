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
	s.requestLogger(r).Errorf("error(status=%d): %s", statusCode, errMsg)

	s.setContentTypeJSON(w)
	w.WriteHeader(statusCode)

	if len(alternativeMessage) > 0 {
		errMsg = strings.Join(alternativeMessage, " ")
	}
	s.writeJSON(w, map[string]string{"error": errMsg})
}

func (s *Server) requestLogger(r *http.Request) *logrus.Entry {
	return s.log.WithField("request_id", middleware.GetReqID(r.Context()))
}
