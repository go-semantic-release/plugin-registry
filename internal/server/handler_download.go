package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-github/v50/github"
)

func (s *Server) getLatestSemRelRelease(ctx context.Context) (*github.RepositoryRelease, error) {
	semrelCacheKey := s.getCacheKeyWithPrefix(cacheKeyPrefixGitHub, "semantic-release/latest")
	cachedLatestRelease, ok := s.getFromCache(semrelCacheKey)
	if ok {
		return cachedLatestRelease.(*github.RepositoryRelease), nil
	}

	err := s.ghSemaphore.Acquire(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("could not acquire semaphore")
	}
	defer s.ghSemaphore.Release(1)

	latestRelease, _, err := s.ghClient.Repositories.GetLatestRelease(ctx, "go-semantic-release", "semantic-release")
	if err != nil {
		return nil, err
	}
	s.setInCache(semrelCacheKey, latestRelease)
	return latestRelease, nil
}

func (s *Server) downloadLatestSemRelBinary(w http.ResponseWriter, r *http.Request) {
	os := chi.URLParam(r, "os")
	arch := chi.URLParam(r, "arch")
	if os == "" || arch == "" {
		s.writeJSONError(w, r, http.StatusBadRequest, fmt.Errorf("missing os or arch"))
		return
	}

	latestRelease, err := s.getLatestSemRelRelease(r.Context())
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not get latest release")
		return
	}

	osArchIdentifier := strings.ToLower(fmt.Sprintf("%s_%s", os, arch))
	for _, asset := range latestRelease.Assets {
		if strings.Contains(asset.GetName(), osArchIdentifier) {
			http.Redirect(w, r, asset.GetBrowserDownloadURL(), http.StatusFound)
			return
		}
	}
	s.writeJSONError(w, r, http.StatusNotFound, fmt.Errorf("could not find binary for  %s/%s", os, arch))
}
