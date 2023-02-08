package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-semantic-release/plugin-registry/internal/config"
)

func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request) {
	res := make([]string, 0)
	for _, p := range config.Plugins {
		res = append(res, p.GetFullName())
	}
	s.writeJSON(w, res)
}

func (s *Server) updateAllPlugins(w http.ResponseWriter, r *http.Request) {
	err := s.ghSemaphore.Acquire(r.Context(), 1)
	if err != nil {
		s.writeJSONError(w, r, http.StatusTooManyRequests, err, "could not acquire semaphore")
		return
	}
	defer s.ghSemaphore.Release(1)

	reqLogger := s.requestLogger(r)
	reqLogger.Warn("updating all plugins...")
	for _, p := range config.Plugins {
		reqLogger.Infof("updating plugin %s", p.GetFullName())
		err := p.Update(r.Context(), s.db, s.ghClient, "")
		if err != nil {
			s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not update plugin")
			return
		}
	}

	s.invalidateByPrefix(s.getCacheKeyPrefixFromPluginName(""))
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) updatePlugin(w http.ResponseWriter, r *http.Request) {
	pluginVersion := chi.URLParam(r, "version")
	pluginName := chi.URLParam(r, "plugin")
	if pluginName == "" {
		s.writeJSONError(w, r, http.StatusBadRequest, fmt.Errorf("plugin name is missing"))
		return
	}
	p := config.Plugins.Find(pluginName)
	if p == nil {
		s.writeJSONError(w, r, http.StatusNotFound, fmt.Errorf("plugin %s not found", pluginName))
		return
	}
	reqLogger := s.requestLogger(r)
	reqLogger.Infof("updating plugin %s@%s", p.GetFullName(), pluginVersion)

	err := s.ghSemaphore.Acquire(r.Context(), 1)
	if err != nil {
		s.writeJSONError(w, r, http.StatusTooManyRequests, err, "could not acquire semaphore")
		return
	}
	defer s.ghSemaphore.Release(1)

	if err := p.Update(r.Context(), s.db, s.ghClient, pluginVersion); err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not update plugin")
		return
	}

	s.invalidateByPrefix(s.getCacheKeyPrefixFromPluginName(p.GetFullName()))
	s.writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) getPlugin(w http.ResponseWriter, r *http.Request) {
	pluginVersion := chi.URLParam(r, "version")
	pluginName := chi.URLParam(r, "plugin")
	if pluginName == "" {
		s.writeJSONError(w, r, http.StatusBadRequest, fmt.Errorf("plugin name is missing"))
		return
	}
	p := config.Plugins.Find(pluginName)
	if p == nil {
		s.writeJSONError(w, r, http.StatusNotFound, fmt.Errorf("plugin %s not found", pluginName))
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

	s.setInCache(r.Context(), s.getCacheKeyFromRequest(r), res)
	s.writeJSON(w, res)
}

func (s *Server) listPluginVersions(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	p := config.Plugins.Find(pluginName)
	if p == nil {
		s.writeJSONError(w, r, http.StatusNotFound, fmt.Errorf("plugin %s not found", pluginName))
		return
	}

	versions, err := p.GetVersions(r.Context(), s.db)
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not get plugin versions")
		return
	}

	s.setInCache(r.Context(), s.getCacheKeyFromRequest(r), versions)
	s.writeJSON(w, versions)
}
