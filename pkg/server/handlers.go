package server

import (
	"fmt"
	"net/http"

	"github.com/go-semantic-release/plugin-registry/pkg/config"
	"github.com/go-semantic-release/plugin-registry/pkg/data"
	"github.com/go-semantic-release/plugin-registry/pkg/plugin"
	"github.com/julienschmidt/httprouter"
)

func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := make([]string, 0)
	for k := range config.PluginMap {
		res = append(res, k)
	}
	s.writeJSON(w, res)
}

func (s *Server) getPluginFromRequest(w http.ResponseWriter, ps httprouter.Params) *plugin.Plugin {
	pluginName := ps.ByName("plugin")
	p, ok := config.PluginMap[pluginName]
	if !ok {
		w.WriteHeader(404)
		s.writeJSONError(w, fmt.Sprintf("plugin %s not found", pluginName))
		return nil
	}
	return p
}

func (s *Server) updatePlugin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pluginVersion := ps.ByName("version")
	p := s.getPluginFromRequest(w, ps)
	if p == nil {
		return
	}
	if err := p.Update(r.Context(), s.db, s.ghClient, pluginVersion); err != nil {
		s.log.Error(err)
		w.WriteHeader(500)
		s.writeJSONError(w, "could not update plugin")
		return
	}
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) getPlugin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pluginVersion := ps.ByName("version")
	p := s.getPluginFromRequest(w, ps)
	if p == nil {
		return
	}
	var err error
	var res interface{}
	if pluginVersion == "" {
		res, err = data.GetPlugin(r.Context(), s.db, p.GetName())
	} else {
		res, err = data.GetPluginRelease(r.Context(), s.db, p.GetName(), pluginVersion)
	}
	if err != nil {
		s.log.Error(err)
		w.WriteHeader(500)
		s.writeJSONError(w, "could not get plugin")
		return
	}
	s.writeJSON(w, res)
}
