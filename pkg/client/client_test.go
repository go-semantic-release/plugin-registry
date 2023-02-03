package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-semantic-release/plugin-registry/pkg/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlugins(t *testing.T) {
	testData := []string{"plugin1", "plugin2"}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/plugins", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(testData))
	}))
	defer ts.Close()
	c := New(ts.URL)
	plugins, err := c.GetPlugins(context.Background())
	require.NoError(t, err)
	require.Equal(t, testData, plugins)
}

func TestGetPluginRelease(t *testing.T) {
	testData := &registry.PluginRelease{
		Version: "1.0.0",
		Assets: map[string]*registry.PluginAsset{
			"darwin/amd64": {
				FileName: "plugin-darwin-amd64",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/plugins/plugin1/versions/1.0.0", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(testData))
	}))
	defer ts.Close()
	c := New(ts.URL)
	pluginRelease, err := c.GetPluginRelease(context.Background(), "plugin1", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "1.0.0", pluginRelease.Version)
	require.Equal(t, "plugin-darwin-amd64", pluginRelease.Assets["darwin/amd64"].FileName)
}

func TestSendBatchRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v2/plugins/_batch", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(&registry.BatchResponse{
			OS:   "darwin",
			Arch: "amd64",
			Plugins: registry.BatchResponsePlugins{
				{
					BatchRequestPlugin: &registry.BatchRequestPlugin{
						FullName:          "plugin1",
						VersionConstraint: "^1.0.0",
					},
					FileName: "plugin1-darwin-amd64",
					URL:      "https://download.example.com/plugin1-darwin-amd64",
				},
			},
		}))
	}))
	defer ts.Close()
	c := New(ts.URL)
	batchResponse, err := c.SendBatchRequest(context.Background(), &registry.BatchRequest{
		OS:   "darwin",
		Arch: "amd64",
		Plugins: []*registry.BatchRequestPlugin{
			{FullName: "plugin1", VersionConstraint: "^1.0.0"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "darwin", batchResponse.OS)
	require.Equal(t, "amd64", batchResponse.Arch)
}

func TestUpdatePlugins(t *testing.T) {
	reqCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "admin-token", r.Header.Get("Authorization"))
		switch reqCount {
		case 0:
			assert.Equal(t, "/api/v2/plugins", r.URL.Path)
		case 1:
			assert.Equal(t, "/api/v2/plugins/provider-git", r.URL.Path)
		case 2:
			assert.Equal(t, "/api/v2/plugins/provider-git/versions/2.0.0", r.URL.Path)
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]bool{"ok": true}))
		reqCount++
	}))
	defer ts.Close()
	c := New(ts.URL)

	err := c.UpdatePlugins(context.Background(), "admin-token")
	require.NoError(t, err)

	err = c.UpdatePlugin(context.Background(), "admin-token", "provider-git")
	require.NoError(t, err)

	err = c.UpdatePluginRelease(context.Background(), "admin-token", "provider-git", "2.0.0")
	require.NoError(t, err)

	require.Equal(t, 3, reqCount)
}
