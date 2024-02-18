package batch

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"github.com/stretchr/testify/require"
)

var (
	testFile         = []byte("test-file")
	testFileChecksum = "3fa65313f3ee7c23d31896e7f57af67618b88dff00f6eb7c3aba2d968d6d4b32"
)

func getTestServer(t *testing.T, failingRequests int) *httptest.Server {
	cnt := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cnt++
		if cnt <= failingRequests {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err := w.Write(testFile)
		require.NoError(t, err)
	}))
}

func TestDownloadFileAndVerifyChecksum(t *testing.T) {
	ts := getTestServer(t, 0)
	defer ts.Close()

	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	err := downloadFileAndVerifyChecksum(context.Background(), tarWriter, "test", ts.URL, testFileChecksum)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())

	tarReader := tar.NewReader(&tarBuffer)
	tarHeader, err := tarReader.Next()
	require.NoError(t, err)
	require.Equal(t, "test", tarHeader.Name)
	require.Equal(t, int64(len(testFile)), tarHeader.Size)
	fileContent, err := io.ReadAll(tarReader)
	require.NoError(t, err)
	require.Equal(t, testFile, fileContent)
}

func TestDownloadFileAndVerifyChecksumRetry(t *testing.T) {
	ts := getTestServer(t, 1)
	defer ts.Close()

	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	err := downloadFileAndVerifyChecksum(context.Background(), tarWriter, "test", ts.URL, testFileChecksum)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())

	tarReader := tar.NewReader(&tarBuffer)
	tarHeader, err := tarReader.Next()
	require.NoError(t, err)
	require.Equal(t, "test", tarHeader.Name)
	require.Equal(t, int64(len(testFile)), tarHeader.Size)
	fileContent, err := io.ReadAll(tarReader)
	require.NoError(t, err)
	require.Equal(t, testFile, fileContent)
}

func createBatchResponsePlugin(url string, i int) *registry.BatchResponsePlugin {
	return &registry.BatchResponsePlugin{
		BatchRequestPlugin: &registry.BatchRequestPlugin{
			FullName: fmt.Sprintf("test-%d", i),
		},
		Version:  "1.0.0",
		FileName: "test",
		URL:      url,
		Checksum: testFileChecksum,
	}
}

func TestDownloadFilesAndTarGz(t *testing.T) {
	ts := getTestServer(t, 0)
	defer ts.Close()

	plugins := make([]*registry.BatchResponsePlugin, 0)
	for i := 0; i < 10; i++ {
		plugins = append(plugins, createBatchResponsePlugin(ts.URL, i))
	}
	batchResponse := &registry.BatchResponse{
		OS:      "linux",
		Arch:    "amd64",
		Plugins: plugins,
	}

	tgzFileName, tgzChecksum, err := DownloadFilesAndTarGz(context.Background(), batchResponse)
	require.NoError(t, err)
	require.NotEmpty(t, tgzFileName)
	defer os.Remove(tgzFileName)

	tgzFile, err := os.ReadFile(tgzFileName)
	require.NoError(t, err)

	tarHash := sha256.New()
	tarHash.Write(tgzFile)
	require.Equal(t, tgzChecksum, hex.EncodeToString(tarHash.Sum(nil)))

	gzipReader, err := gzip.NewReader(bytes.NewReader(tgzFile))
	require.NoError(t, err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for i := 0; i < 10; i++ {
		tarHeader, err := tarReader.Next()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("linux_amd64/test-%d/1.0.0/test", i), tarHeader.Name)
		require.Equal(t, int64(len(testFile)), tarHeader.Size)

		fileContent, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		require.Equal(t, testFile, fileContent)
	}
}
