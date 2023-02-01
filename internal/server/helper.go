package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) writeJSON(w http.ResponseWriter, d any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(d)
	if err != nil {
		s.log.Error(err)
	}
}

func (s *Server) writeJSONError(w http.ResponseWriter, r *http.Request, statusCode int, err error, alternativeMessage ...string) {
	errMsg := err.Error()
	s.log.Errorf("[%s] error(status=%d): %s", middleware.GetReqID(r.Context()), statusCode, errMsg)
	w.WriteHeader(statusCode)
	if len(alternativeMessage) > 0 {
		errMsg = strings.Join(alternativeMessage, " ")
	}
	s.writeJSON(w, map[string]string{"error": errMsg})
}
