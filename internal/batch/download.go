package batch

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	defaultRetryableClient     *retryablehttp.Client
	defaultRetryableClientInit sync.Once
)

func getDefaultRetryableClient() *retryablehttp.Client {
	defaultRetryableClientInit.Do(func() {
		defaultRetryableClient = retryablehttp.NewClient()
		defaultRetryableClient.Logger = nil
		defaultRetryableClient.HTTPClient.Timeout = 3 * time.Minute
	})
	return defaultRetryableClient
}

func downloadFileAndVerifyChecksum(ctx context.Context, tarWriter *tar.Writer, fileName, url, checksum string) error {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := getDefaultRetryableClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	err = tarWriter.WriteHeader(&tar.Header{
		Name: fileName,
		Mode: 0o755,
		Size: resp.ContentLength,
	})
	if err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	checksumReader := io.TeeReader(resp.Body, tarWriter)
	checksumHash := sha256.New()
	n, err := io.Copy(checksumHash, checksumReader)
	if err != nil {
		return fmt.Errorf("failed to write tar file: %w", err)
	}
	if n != resp.ContentLength {
		return fmt.Errorf("unexpected content length: %d (should be %d)", n, resp.ContentLength)
	}
	if checksum != "" && hex.EncodeToString(checksumHash.Sum(nil)) != checksum {
		return fmt.Errorf("checksum verification failed")
	}
	return nil
}

func DownloadFilesAndTarGz(ctx context.Context, batchResponse *registry.BatchResponse) (string, string, error) {
	tgzFile, err := os.CreateTemp("", "plugin-archive-*.tar.gz")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tgzFile.Close()

	tgzHash := sha256.New()
	gzipWriter := gzip.NewWriter(io.MultiWriter(tgzFile, tgzHash))
	tarWriter := tar.NewWriter(gzipWriter)
	for _, plugin := range batchResponse.Plugins {
		fileName := fmt.Sprintf("%s_%s/%s/%s/%s", batchResponse.OS, batchResponse.Arch, plugin.FullName, plugin.Version, plugin.FileName)
		err = downloadFileAndVerifyChecksum(ctx, tarWriter, fileName, plugin.URL, plugin.Checksum)
		if err != nil {
			return "", "", fmt.Errorf("failed to add file to tar archive: %w", err)
		}
	}
	err = tarWriter.Close()
	if err != nil {
		return "", "", fmt.Errorf("failed to close tar writer: %w", err)
	}
	err = gzipWriter.Close()
	if err != nil {
		return "", "", fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return tgzFile.Name(), hex.EncodeToString(tgzHash.Sum(nil)), nil
}
