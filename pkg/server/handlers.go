package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/go-chi/chi/v5"
	"github.com/go-semantic-release/plugin-registry/pkg/config"
	"github.com/go-semantic-release/plugin-registry/pkg/registry"
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

func validateAndCreatePluginResponses(batchRequest *registry.BatchRequest) (registry.BatchPluginResponses, error) {
	err := batchRequest.Validate()
	if err != nil {
		return nil, err
	}
	pluginResponses := make(registry.BatchPluginResponses, 0)
	for _, pluginReq := range batchRequest.Plugins {
		if !strings.Contains(pluginReq.FullName, "-") {
			return nil, fmt.Errorf("plugin %s has an invalid name", pluginReq.FullName)
		}

		if pluginReq.VersionConstraint == "" {
			pluginReq.VersionConstraint = "latest"
		}
		if pluginReq.VersionConstraint != "latest" {
			versionConstraint, err := semver.NewConstraint(pluginReq.VersionConstraint)
			if err != nil {
				return nil, fmt.Errorf("plugin %s has an invalid version constraint", pluginReq.FullName)
			}
			pluginReq.VersionConstraint = versionConstraint.String()
		}

		if pluginResponses.Has(pluginReq.FullName) {
			return nil, fmt.Errorf("plugin %s requested multiple times", pluginReq.FullName)
		}

		p := config.Plugins.Find(pluginReq.FullName)
		if p == nil {
			return nil, fmt.Errorf("plugin %s does not exist", pluginReq.FullName)
		}

		pluginResponses = append(pluginResponses, registry.NewBatchPluginResponse(pluginReq))
	}
	return pluginResponses, nil
}

func (s *Server) batchGetPlugins(w http.ResponseWriter, r *http.Request) {
	batchRequest := new(registry.BatchRequest)
	if err := json.NewDecoder(r.Body).Decode(batchRequest); err != nil {
		s.writeJSONError(w, r, http.StatusBadRequest, err, "could not decode request")
		return
	}

	pluginResponses, err := validateAndCreatePluginResponses(batchRequest)
	if err != nil {
		s.writeJSONError(w, r, http.StatusBadRequest, err)
		return
	}

	batchResponse := registry.NewBatchResponse(batchRequest, pluginResponses)

	for _, pluginResponse := range batchResponse.Plugins {
		p := config.Plugins.Find(pluginResponse.FullName)
		foundRelease, err := p.GetReleaseWithVersionConstraint(r.Context(), s.db, pluginResponse.VersionConstraint)
		if err != nil {
			s.writeJSONError(w, r, http.StatusBadRequest, err, fmt.Sprintf("could not resolve plugin %s", pluginResponse.FullName))
			return
		}
		foundAsset := foundRelease.Assets[batchResponse.GetOSArch()]
		if foundAsset == nil {
			s.writeJSONError(w, r, http.StatusBadRequest, fmt.Errorf("could not find %s asset for plugin %s", batchResponse.GetOSArch(), pluginResponse.FullName))
			return
		}

		pluginResponse.Version = foundRelease.Version
		pluginResponse.FileName = foundAsset.FileName
		pluginResponse.URL = foundAsset.URL
		pluginResponse.Checksum = foundAsset.Checksum
	}

	batchResponse.Hash()

	s.writeJSON(w, batchResponse)
}
