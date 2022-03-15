package server

import (
	"net/http"

	"github.com/go-semantic-release/plugin-registry/pkg/config"
	"github.com/julienschmidt/httprouter"
)

func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	res := make([]string, 0)
	for k := range config.PluginMap {
		res = append(res, k)
	}
	s.writeJSON(w, res)
}
