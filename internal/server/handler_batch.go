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
	"github.com/go-semantic-release/plugin-registry/internal/batch"
	"github.com/go-semantic-release/plugin-registry/internal/config"
	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"golang.org/x/sync/errgroup"
)

type pluginBatchError struct {
	PluginName string
	Err        error
}

func (e *pluginBatchError) Error() string {
	return fmt.Sprintf("plugin batch error (%s): %s", e.PluginName, e.Err.Error())
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
	// limit request body to 1MB
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
	batchRequestCacheKey := s.getCacheKeyWithPrefix(cacheKeyPrefixBatchRequest, hex.EncodeToString(batchResponse.Hash()))
	cachedBatchResponse, found := s.getFromCache(batchRequestCacheKey)
	if found {
		s.log.Infof("found cached batch response for %s", batchRequestCacheKey)
		s.writeJSON(w, cachedBatchResponse)
		return
	}

	// TODO: this could be done in parallel
	errGroup, groupCtx := errgroup.WithContext(r.Context())
	errGroup.SetLimit(5)
	for _, pluginResponse := range batchResponse.Plugins {
		pluginResponse := pluginResponse
		errGroup.Go(func() error {
			p := config.Plugins.Find(pluginResponse.FullName)
			foundRelease, rErr := p.GetReleaseWithVersionConstraint(groupCtx, s.db, pluginResponse.VersionConstraint)
			if rErr != nil {
				return &pluginBatchError{
					PluginName: pluginResponse.FullName,
					Err:        rErr,
				}
			}
			foundAsset := foundRelease.Assets[batchResponse.GetOSArch()]
			if foundAsset == nil {
				return &pluginBatchError{
					PluginName: pluginResponse.FullName,
					Err:        fmt.Errorf("could not find %s asset", batchResponse.GetOSArch()),
				}
			}
			pluginResponse.Version = foundRelease.Version
			pluginResponse.FileName = foundAsset.FileName
			pluginResponse.URL = foundAsset.URL
			pluginResponse.Checksum = foundAsset.Checksum
			return nil
		})
	}
	err = errGroup.Wait()
	pbErr := &pluginBatchError{}
	if errors.As(err, &pbErr) {
		s.writeJSONError(w, r, http.StatusBadRequest, pbErr, fmt.Sprintf("could not resolve plugin %s", pbErr.PluginName))
		return
	} else if err != nil {
		s.writeJSONError(w, r, http.StatusBadRequest, err, "could not resolve plugins")
		return
	}

	// calculate the hash of the response, this now includes the plugin versions
	batchResponse.CalculateHash()
	archiveKey := fmt.Sprintf("archives/plugins-%s.tar.gz", batchResponse.DownloadHash)
	// the download url is deterministic, so we can set it here
	batchResponse.DownloadURL = s.config.GetPublicPluginCacheDownloadURL(archiveKey)

	// allow only one batch archive process at a time
	// TODO: this might be to conservative, we could allow multiple archives to be created at the same time
	s.batchMu.Lock()
	defer s.batchMu.Unlock()

	headRes, err := s.storage.HeadObject(r.Context(), &s3.HeadObjectInput{
		Bucket: s.config.GetBucket(),
		Key:    &archiveKey,
	})
	if err == nil {
		// the archive already exists, return the response
		s.log.Infof("found cached archive %s", archiveKey)
		batchResponse.DownloadChecksum = headRes.Metadata["checksum"]
		s.setInCache(batchRequestCacheKey, batchResponse)
		s.writeJSON(w, batchResponse)
		return
	}

	var genericAPIError *smithy.GenericAPIError
	if !errors.As(err, &genericAPIError) || genericAPIError.ErrorCode() != "NotFound" {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not check if plugin archive exists")
		return
	}

	s.log.Infof("plugin archive %s not found, creating...", archiveKey)
	tgzFileName, tgzChecksum, err := batch.DownloadFilesAndTarGz(r.Context(), batchResponse)
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not create plugin archive")
		return
	}
	batchResponse.DownloadChecksum = tgzChecksum
	s.log.Infof("created plugin archive %s, uploading...", tgzFileName)
	tarFile, err := os.Open(tgzFileName)
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not open plugin archive")
		return
	}

	_, err = s.storage.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      s.config.GetBucket(),
		Key:         &archiveKey,
		Body:        tarFile,
		ContentType: aws.String("application/gzip"),
		Metadata: map[string]string{
			"checksum": tgzChecksum,
			"hash":     batchResponse.DownloadHash,
		},
	})
	if closeErr := tarFile.Close(); closeErr != nil {
		s.log.Errorf("could not close plugin archive file: %v", closeErr)
	}
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not upload plugin archive")
		return
	}

	s.log.Infof("uploaded plugin archive.")
	if rmErr := os.Remove(tgzFileName); rmErr != nil {
		s.log.Errorf("could not remove plugin archive file: %v", rmErr)
	}

	s.setInCache(batchRequestCacheKey, batchResponse)
	s.writeJSON(w, batchResponse)
}
