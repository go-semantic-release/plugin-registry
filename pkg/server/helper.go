package server

import (
	"encoding/json"
	"io"
	"net/http"
)

func (s *Server) writeJSON(w io.Writer, d any) {
	err := json.NewEncoder(w).Encode(d)
	if err != nil {
		s.log.Error(err)
	}
}

func (s *Server) writeJSONError(w http.ResponseWriter, statusCode int, err error, msg string) {
	if err != nil {
		s.log.Errorf("ERROR(status=%d): %v", statusCode, err)
	}
	w.WriteHeader(statusCode)
	s.writeJSON(w, map[string]string{"error": msg})
}
