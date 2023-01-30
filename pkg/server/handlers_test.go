package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-semantic-release/plugin-registry/pkg/config"
	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"github.com/google/go-github/v50/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newGitHubClient() *github.Client {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposReleasesByOwnerByRepo,
			[]*github.RepositoryRelease{
				{Draft: github.Bool(false), TagName: github.String("v1.0.0")},
				{Draft: github.Bool(false), TagName: github.String("v1.0.1")},
				{Draft: github.Bool(true), TagName: github.String("v2.0.0-beta")},
				{
					Draft:   github.Bool(false),
					TagName: github.String("v2.0.0"),
					Assets: []*github.ReleaseAsset{
						{
							Name:               github.String("condition-default_darwin_amd64"),
							BrowserDownloadURL: github.String("https://download.example/condition-default_darwin_amd64"),
						},
						{
							Name:               github.String("condition-default_linux_amd64"),
							BrowserDownloadURL: github.String("https://download.example/condition-default_linux_amd64"),
						},
					},
				},
			},
		),
		mock.WithRequestMatch(
			mock.GetReposReleasesLatestByOwnerByRepo,
			&github.RepositoryRelease{
				TagName: github.String("v2.0.0"),
			},
			&github.RepositoryRelease{
				TagName: github.String("v3.0.0"),
			},
		),
		mock.WithRequestMatch(
			mock.GetReposReleasesTagsByOwnerByRepoByTag,
			&github.RepositoryRelease{
				Draft:   github.Bool(false),
				TagName: github.String("v3.0.0"),
				Assets: []*github.ReleaseAsset{
					{
						Name:               github.String("condition-default_darwin_arm64"),
						BrowserDownloadURL: github.String("https://download.example/condition-default_darwin_arm64"),
					},
				},
			},
		),
	)
	return github.NewClient(mockedHTTPClient)
}

// adapted from https://www.captaincodeman.com/unit-testing-with-firestore-emulator-and-go
func starsFirebaseEmulator() (func(), error) {
	cmd := exec.Command("gcloud", "emulators", "firestore", "start", "--host-port=127.0.0.1:9090")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	killFn := func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		signal.Stop(sigCh)
	}
	go func() {
		<-sigCh
		killFn()
	}()
	return killFn, nil
}

func newTestServer() (*Server, *firestore.Client, error) {
	log := logrus.New()
	log.Out = io.Discard

	_ = os.Setenv("FIRESTORE_EMULATOR_HOST", "127.0.0.1:9090")
	_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	fsClient, err := firestore.NewClient(context.Background(), "go-semantic-release")
	if err != nil {
		return nil, nil, err
	}

	return New(log, fsClient, newGitHubClient(), "admin-token"), fsClient, nil
}

func sendRequest(s http.Handler, method, path string, body io.Reader, modReqFns ...func(req *http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	for _, f := range modReqFns {
		f(req)
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func TestListPlugins(t *testing.T) {
	s, _, err := newTestServer()
	require.NoError(t, err)

	rr := sendRequest(s, "GET", "/api/v2/plugins", nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var plugins []string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &plugins))
	require.Len(t, plugins, len(config.Plugins))
}

func saveDoc(fsClient *firestore.Client, collection, doc string, data map[string]any) error {
	_, err := fsClient.Collection(collection).Doc(doc).Set(context.Background(), data)
	return err
}

func createPluginDoc(fsClient *firestore.Client, fullName, latestRelease string) error {
	pluginType, name, _ := strings.Cut(fullName, "-")
	err := saveDoc(fsClient, "plugins", fullName, map[string]any{
		"FullName":         fullName,
		"Type":             pluginType,
		"Name":             name,
		"URL":              fmt.Sprintf("https://github.com/my-org/%s", fullName),
		"LatestReleaseRef": fsClient.Doc(fmt.Sprintf("plugins/%s/versions/%s", fullName, latestRelease)),
	})
	if err != nil {
		return err
	}

	versionsCollection := fmt.Sprintf("plugins/%s/versions", fullName)

	for _, version := range []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0", "3.0.0", latestRelease} {
		err = saveDoc(fsClient, versionsCollection, version, map[string]any{
			"Version":    version,
			"Prerelease": false,
			"CreatedAt":  time.Now(),
			"Assets": map[string]map[string]string{
				"darwin/amd64": {
					"FileName": fullName + "-darwin-amd64",
					"URL":      "https//download.example.com/" + fullName + "-darwin-amd64",
					"OS":       "darwin",
					"Arch":     "amd64",
					"Checksum": "1234",
				},
				"linux/amd64": {
					"FileName": fullName + "-linux-amd64",
					"URL":      "https//download.example.com/" + fullName + "-linux-amd64",
					"OS":       "linux",
					"Arch":     "amd64",
					"Checksum": "5678",
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return err
}

func bootstrapDatabase(fsClient *firestore.Client) error {
	err := createPluginDoc(fsClient, "provider-git", "3.0.0")
	if err != nil {
		return err
	}
	err = createPluginDoc(fsClient, "condition-github", "4.0.0")
	if err != nil {
		return err
	}

	err = createPluginDoc(fsClient, "hooks-goreleaser", "5.0.0")
	if err != nil {
		return err
	}

	return err
}

func TestGetPlugin(t *testing.T) {
	killFirebaseEmulator, err := starsFirebaseEmulator()
	require.NoError(t, err)
	defer killFirebaseEmulator()
	s, fsClient, err := newTestServer()
	require.NoError(t, err)

	require.NoError(t, bootstrapDatabase(fsClient))

	rr := sendRequest(s, "GET", "/api/v2/plugins/provider-git", nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var plugin registry.Plugin
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &plugin))
	require.Len(t, plugin.Versions, 5)
	require.Equal(t, "3.0.0", plugin.LatestRelease.Version)
	require.Equal(t, "provider-git-darwin-amd64", plugin.LatestRelease.Assets["darwin/amd64"].FileName)
}

func TestUpdateAndGetPlugin(t *testing.T) {
	killFirebaseEmulator, err := starsFirebaseEmulator()
	require.NoError(t, err)
	defer killFirebaseEmulator()
	s, _, err := newTestServer()
	require.NoError(t, err)

	rr := sendRequest(s, "PUT", "/api/v2/plugins/condition-default", bytes.NewBufferString("{}"), func(req *http.Request) {
		req.Header.Set("Authorization", "admin-token")
	})
	require.Equal(t, http.StatusOK, rr.Code)

	rr = sendRequest(s, "GET", "/api/v2/plugins/condition-default", nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var plugin registry.Plugin
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &plugin))
	require.Len(t, plugin.Versions, 3)
	require.Equal(t, "2.0.0", plugin.LatestRelease.Version)
	require.Len(t, plugin.LatestRelease.Assets, 2)
	require.Equal(t, "condition-default_darwin_amd64", plugin.LatestRelease.Assets["darwin/amd64"].FileName)

	rr = sendRequest(s, "PUT", "/api/v2/plugins/condition-default/versions/3.0.0", bytes.NewBufferString("{}"), func(req *http.Request) {
		req.Header.Set("Authorization", "admin-token")
	})
	require.Equal(t, http.StatusOK, rr.Code)

	rr = sendRequest(s, "GET", "/api/v2/plugins/condition-default", nil)
	require.Equal(t, http.StatusOK, rr.Code)
	plugin = registry.Plugin{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &plugin))
	require.Len(t, plugin.Versions, 4)
	require.Equal(t, "3.0.0", plugin.LatestRelease.Version)
	require.Len(t, plugin.LatestRelease.Assets, 1)
	require.Equal(t, "condition-default_darwin_arm64", plugin.LatestRelease.Assets["darwin/arm64"].FileName)
}

func sendBatchRequest(t *testing.T, s *Server, br *registry.BatchRequest) *httptest.ResponseRecorder {
	var bodyBuffer bytes.Buffer
	require.NoError(t, json.NewEncoder(&bodyBuffer).Encode(br))
	rr := sendRequest(s, "POST", "/api/v2/plugins/_batch", &bodyBuffer)
	return rr
}

func TestBatchEndpoint(t *testing.T) {
	killFirebaseEmulator, err := starsFirebaseEmulator()
	require.NoError(t, err)
	defer killFirebaseEmulator()
	s, fsClient, err := newTestServer()
	require.NoError(t, err)

	require.NoError(t, bootstrapDatabase(fsClient))

	batchRequest := &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "condition-github", VersionConstraint: "latest"},
			{FullName: "hooks-goreleaser", VersionConstraint: ""},
			{FullName: "provider-git", VersionConstraint: "^1.0.0"},
		},
	}

	rr := sendBatchRequest(t, s, batchRequest)
	require.Equal(t, http.StatusOK, rr.Code)
	var batchResponse registry.BatchResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &batchResponse))
	require.Len(t, batchResponse.Plugins, 3)
	require.Equal(t, "4.0.0", batchResponse.Plugins[0].Version)
	require.Equal(t, "latest", batchResponse.Plugins[0].VersionConstraint)
	require.Equal(t, "5.0.0", batchResponse.Plugins[1].Version)
	require.Equal(t, "latest", batchResponse.Plugins[1].VersionConstraint)
	require.Equal(t, "1.2.0", batchResponse.Plugins[2].Version)
	require.Equal(t, "^1.0.0", batchResponse.Plugins[2].VersionConstraint)
	require.Equal(t, "f3a53717f71bc03b4a784eba5dd1f2454edc4c418a291ae038446236cb559611", batchResponse.DownloadHash)
}

func decodeError(t *testing.T, body []byte) string {
	var err struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &err))
	return err.Error
}

func TestBatchEndpointBadRequests(t *testing.T) {
	killFirebaseEmulator, err := starsFirebaseEmulator()
	require.NoError(t, err)
	defer killFirebaseEmulator()
	s, fsClient, err := newTestServer()
	require.NoError(t, err)

	require.NoError(t, bootstrapDatabase(fsClient))

	rr := sendBatchRequest(t, s, &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "wrong", VersionConstraint: "latest"},
		},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, decodeError(t, rr.Body.Bytes()), "has an invalid name")

	rr = sendBatchRequest(t, s, &registry.BatchRequest{
		OS:      "darwin",
		Arch:    "amd64",
		Plugins: []*registry.BatchRequestPlugin{},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, decodeError(t, rr.Body.Bytes()), "at least one plugin is required")

	rr = sendBatchRequest(t, s, &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "provider-git", VersionConstraint: "xxxxxxx"},
		},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, decodeError(t, rr.Body.Bytes()), "invalid version constraint")

	rr = sendBatchRequest(t, s, &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "provider-giiiit", VersionConstraint: "latest"},
		},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, decodeError(t, rr.Body.Bytes()), "does not exist")

	rr = sendBatchRequest(t, s, &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "provider-git", VersionConstraint: "latest"},
			{FullName: "provider-git", VersionConstraint: "latest"},
		},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, decodeError(t, rr.Body.Bytes()), "requested multiple times")

	rr = sendBatchRequest(t, s, &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "provider-gitlab", VersionConstraint: "^8.0.0"},
		},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, decodeError(t, rr.Body.Bytes()), "could not resolve")
}
