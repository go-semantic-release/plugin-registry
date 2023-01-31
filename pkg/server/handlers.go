package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-semantic-release/plugin-registry/pkg/batch"
	"github.com/go-semantic-release/plugin-registry/pkg/config"
	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"github.com/patrickmn/go-cache"
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

func validateAndCreatePluginResponses(batchRequest *registry.BatchRequest) (registry.BatchResponsePlugins, error) {
	err := batchRequest.Validate()
	if err != nil {
		return nil, err
	}
	pluginResponses := make(registry.BatchResponsePlugins, 0)
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

		pluginResponses = append(pluginResponses, registry.NewBatchResponsePlugin(pluginReq))
	}
	return pluginResponses, nil
}

//gocyclo:ignore
func (s *Server) batchGetPlugins(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

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

	// hash the batch request without the resolved versions
	batchRequestHash := hex.EncodeToString(batchResponse.Hash())
	cachedBatchResponse, found := s.cache.Get(batchRequestHash)
	if found {
		s.log.Infof("found cached batch response for %s", batchRequestHash)
		s.writeJSON(w, cachedBatchResponse)
		return
	}

	for _, pluginResponse := range batchResponse.Plugins {
		p := config.Plugins.Find(pluginResponse.FullName)
		foundRelease, rErr := p.GetReleaseWithVersionConstraint(r.Context(), s.db, pluginResponse.VersionConstraint)
		if rErr != nil {
			s.writeJSONError(w, r, http.StatusBadRequest, rErr, fmt.Sprintf("could not resolve plugin %s", pluginResponse.FullName))
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

	// calculate the hash of the response, this now includes the plugin versions
	batchResponse.CalculateHash()
	archiveKey := fmt.Sprintf("archives/plugins-%s.tar.gz", batchResponse.DownloadHash)
	// the download url is deterministic, so we can set it here
	batchResponse.DownloadURL = s.config.GetPublicPluginCacheDownloadURL(archiveKey)

	_, err = s.storage.HeadObject(r.Context(), &s3.HeadObjectInput{
		Bucket: s.config.GetBucket(),
		Key:    &archiveKey,
	})
	if err == nil {
		// the archive already exists, return the response
		s.log.Infof("found cached archive %s", archiveKey)
		s.cache.Set(batchRequestHash, batchResponse, cache.DefaultExpiration)
		s.writeJSON(w, batchResponse)
		return
	}

	var genericAPIError *smithy.GenericAPIError
	if !errors.As(err, &genericAPIError) || genericAPIError.ErrorCode() != "NotFound" {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not check if plugin archive exists")
		return
	}

	s.log.Infof("plugin archive %s not found, creating...", archiveKey)
	tarFileName, err := batch.DownloadFilesAndTarGz(r.Context(), batchResponse)
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not create plugin archive")
		return
	}
	s.log.Infof("created plugin archive %s, uploading...", tarFileName)
	tarFile, err := os.Open(tarFileName)
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not open plugin archive")
		return
	}

	putRes, err := s.storage.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      s.config.GetBucket(),
		Key:         &archiveKey,
		Body:        tarFile,
		ContentType: aws.String("application/gzip"),
	})
	if closeErr := tarFile.Close(); closeErr != nil {
		s.log.Errorf("could not close plugin archive file: %v", closeErr)
	}
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not upload plugin archive")
		return
	}
	s.log.Infof("uploaded plugin archive %s", *putRes.VersionId)
	if rmErr := os.Remove(tarFileName); rmErr != nil {
		s.log.Errorf("could not remove plugin archive file: %v", rmErr)
	}

	s.cache.Set(batchRequestHash, batchResponse, cache.DefaultExpiration)
	s.writeJSON(w, batchResponse)
}
