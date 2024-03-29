package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	reqLogger := s.requestLogger(r)
	batchResponse := registry.NewBatchResponse(batchRequest, pluginResponses)

	// hash the batch request without the resolved versions
	batchRequestCacheKey := s.getCacheKeyWithPrefix(cacheKeyPrefixBatchRequest, hex.EncodeToString(batchResponse.Hash()))
	cachedBatchResponse, found := s.getFromCache(r.Context(), batchRequestCacheKey)
	if found {
		reqLogger.Infof("found cached batch response for %s", batchRequestCacheKey)
		s.writeJSON(w, cachedBatchResponse)
		return
	}

	// resolve plugins
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
	err = s.batchArchiveSemaphore.Acquire(r.Context(), 1)
	if err != nil {
		s.writeJSONError(w, r, http.StatusTooManyRequests, err, "could not acquire semaphore")
		return
	}
	defer s.batchArchiveSemaphore.Release(1)

	headRes, err := s.storage.HeadObject(r.Context(), &s3.HeadObjectInput{
		Bucket: s.config.GetBucket(),
		Key:    &archiveKey,
	})
	if err == nil {
		// the archive already exists, return the response
		reqLogger.Infof("found cached archive %s", archiveKey)
		batchResponse.DownloadChecksum = headRes.Metadata["checksum"]
		s.setInCache(r.Context(), batchRequestCacheKey, batchResponse)
		s.writeJSON(w, batchResponse)
		return
	}

	var s3ResponseError *awshttp.ResponseError
	if !errors.As(err, &s3ResponseError) || s3ResponseError.HTTPStatusCode() != http.StatusNotFound {
		reqLogger.Errorf("could not check if plugin archive exists: %v", err)
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not check if plugin archive exists")
		return
	}

	reqLogger.Infof("plugin archive %s not found, creating (%d plugins for %s)...", archiveKey, len(batchResponse.Plugins), batchResponse.GetOSArch())
	tgzFileName, tgzChecksum, err := batch.DownloadFilesAndTarGz(r.Context(), batchResponse)
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not create plugin archive")
		return
	}
	batchResponse.DownloadChecksum = tgzChecksum
	reqLogger.Infof("created plugin archive %s, uploading...", tgzFileName)
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
			"checksum":  tgzChecksum,
			"hash":      batchResponse.DownloadHash,
			"os":        batchResponse.OS,
			"arch":      batchResponse.Arch,
			"plugins":   strconv.Itoa(len(batchResponse.Plugins)),
			"cache_key": string(batchRequestCacheKey),
		},
	})
	if closeErr := tarFile.Close(); closeErr != nil {
		reqLogger.Errorf("could not close plugin archive file: %v", closeErr)
	}
	if err != nil {
		s.writeJSONError(w, r, http.StatusInternalServerError, err, "could not upload plugin archive")
		return
	}

	reqLogger.Infof("uploaded plugin archive.")
	if rmErr := os.Remove(tgzFileName); rmErr != nil {
		reqLogger.Errorf("could not remove plugin archive file: %v", rmErr)
	}

	s.setInCache(r.Context(), batchRequestCacheKey, batchResponse)
	s.writeJSON(w, batchResponse)
}
