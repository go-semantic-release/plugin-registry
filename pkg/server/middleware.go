package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *Server) authMiddleware(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if s.adminAccessToken == "" {
			s.writeJSONError(w, http.StatusUnauthorized, nil, "no access token configured")
			return
		}
		if r.Header.Get("Authorization") != s.adminAccessToken {
			s.writeJSONError(w, http.StatusUnauthorized, nil, "invalid access token")
			return
		}
		h(w, r, ps)
	}
}
