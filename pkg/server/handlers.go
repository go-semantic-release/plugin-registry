package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-semantic-release/plugin-registry/pkg/config"
)

func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request) {
	res := make([]string, 0)
	for _, p := range config.Plugins {
		res = append(res, p.GetFullName())
	}
	s.writeJSON(w, res)
}

func (s *Server) updateAllPlugins(w http.ResponseWriter, r *http.Request) {
	s.ghMutex.Lock()
	defer s.ghMutex.Unlock()
	s.log.Warn("updating all plugins")
	for _, p := range config.Plugins {
		s.log.Infof("updating plugin %s", p.GetFullName())
		err := p.Update(r.Context(), s.db, s.ghClient, "")
		if err != nil {
			s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not update plugin")
			return
		}
	}
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) updatePlugin(w http.ResponseWriter, r *http.Request) {
	s.ghMutex.Lock()
	defer s.ghMutex.Unlock()
	pluginVersion := chi.URLParam(r, "version")
	pluginName := chi.URLParam(r, "plugin")
	p := config.Plugins.Find(pluginName)
	if p == nil {
		s.writeJSONError(w, r, 404, fmt.Errorf("plugin %s not found", pluginName))
		return
	}
	s.log.Infof("updating plugin %s@%s", p.GetFullName(), pluginVersion)
	if err := p.Update(r.Context(), s.db, s.ghClient, pluginVersion); err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not update plugin")
		return
	}
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) getPlugin(w http.ResponseWriter, r *http.Request) {
	pluginVersion := chi.URLParam(r, "version")
	pluginName := chi.URLParam(r, "plugin")
	p := config.Plugins.Find(pluginName)
	if p == nil {
		s.writeJSONError(w, r, 404, fmt.Errorf("plugin %s not found", pluginName))
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
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not get plugin")
		return
	}
	s.writeJSON(w, res)
}
