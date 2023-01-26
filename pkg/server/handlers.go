package server

import (
	"fmt"
	"net/http"

	"github.com/go-semantic-release/plugin-registry/pkg/config"
	"github.com/go-semantic-release/plugin-registry/pkg/plugin"
	"github.com/julienschmidt/httprouter"
)

func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := make([]string, 0)
	for _, p := range config.Plugins {
		res = append(res, p.GetFullName())
	}
	s.writeJSON(w, res)
}

func (s *Server) updateAllPlugins(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.ghMutex.Lock()
	defer s.ghMutex.Unlock()
	for _, p := range config.Plugins {
		err := p.Update(r.Context(), s.db, s.ghClient, "")
		if err != nil {
			s.write500Error(w, err, "could not update plugin")
			return
		}
	}
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) getPluginFromRequest(w http.ResponseWriter, ps httprouter.Params) *plugin.Plugin {
	pluginName := ps.ByName("plugin")
	p := config.Plugins.Find(pluginName)
	if p == nil {
		w.WriteHeader(404)
		s.writeJSONError(w, fmt.Sprintf("plugin %s not found", pluginName))
		return nil
	}
	return p
}

func (s *Server) updatePlugin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	s.ghMutex.Lock()
	defer s.ghMutex.Unlock()
	pluginVersion := ps.ByName("version")
	p := s.getPluginFromRequest(w, ps)
	if p == nil {
		return
	}
	if err := p.Update(r.Context(), s.db, s.ghClient, pluginVersion); err != nil {
		s.write500Error(w, err, "could not update plugin")
		return
	}
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) getPlugin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pluginVersion := ps.ByName("version")
	p := s.getPluginFromRequest(w, ps)
	if p == nil {
		// getPluginFromRequest already wrote the error
		return
	}
	var err error
	var res any
	if pluginVersion == "" {
		res, err = p.Get(r.Context(), s.db)
	} else {
		res, err = p.GetRelease(r.Context(), s.db, pluginVersion)
	}
	if err != nil {
		s.write500Error(w, err, "could not get plugin")
		return
	}
	s.writeJSON(w, res)
}
