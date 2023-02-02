package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-semantic-release/plugin-registry/pkg/registry"
)

type ErrorResponse struct {
	StatusCode int
	ErrorMsg   string `json:"error"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("unexpected status code: %d, error: %s", e.StatusCode, e.ErrorMsg)
}

type Client struct {
	registryURL string
	httpClient  *http.Client
}

func New(registryURL string) *Client {
	return &Client{
		registryURL: registryURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func setAuth(adminAccessToken string) func(r *http.Request) {
	return func(r *http.Request) {
		r.Header.Set("Authorization", adminAccessToken)
	}
}

func getPluginURL(pluginName string) string {
	return fmt.Sprintf("plugins/%s", pluginName)
}

func getPluginReleaseURL(pluginName, version string) string {
	return fmt.Sprintf("%s/versions/%s", getPluginURL(pluginName), version)
}

func (c *Client) sendRequest(ctx context.Context, method, endpoint string, body io.Reader, modifyRequestFns ...func(r *http.Request)) (*http.Response, error) {
	apiEndpoint, err := url.JoinPath(c.registryURL, endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, apiEndpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	for _, f := range modifyRequestFns {
		f(req)
	}
	return c.httpClient.Do(req)
}

func (c *Client) decodeResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		if err != nil {
			return err
		}
		return &errResp
	}
	err := json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetPlugins(ctx context.Context) ([]string, error) {
	resp, err := c.sendRequest(ctx, http.MethodGet, "plugins", nil)
	if err != nil {
		return nil, err
	}
	var plugins []string
	err = c.decodeResponse(resp, &plugins)
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

func (c *Client) GetPlugin(ctx context.Context, pluginName string) (*registry.Plugin, error) {
	resp, err := c.sendRequest(ctx, http.MethodGet, getPluginURL(pluginName), nil)
	if err != nil {
		return nil, err
	}
	var p registry.Plugin
	err = c.decodeResponse(resp, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) GetPluginRelease(ctx context.Context, pluginName, version string) (*registry.PluginRelease, error) {
	resp, err := c.sendRequest(ctx, http.MethodGet, getPluginReleaseURL(pluginName, version), nil)
	if err != nil {
		return nil, err
	}
	var pr registry.PluginRelease
	err = c.decodeResponse(resp, &pr)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (c *Client) SendBatchRequest(ctx context.Context, batch *registry.BatchRequest) (*registry.BatchResponse, error) {
	var bodyBuffer bytes.Buffer
	err := json.NewEncoder(&bodyBuffer).Encode(batch)
	if err != nil {
		return nil, err
	}
	resp, err := c.sendRequest(ctx, http.MethodPost, "plugins/_batch", &bodyBuffer)
	if err != nil {
		return nil, err
	}
	var br registry.BatchResponse
	err = c.decodeResponse(resp, &br)
	if err != nil {
		return nil, err
	}
	return &br, nil
}

func (c *Client) UpdatePlugins(ctx context.Context, adminAccessToken string) error {
	return c.UpdatePluginRelease(ctx, adminAccessToken, "", "")
}

func (c *Client) UpdatePlugin(ctx context.Context, adminAccessToken, pluginName string) error {
	return c.UpdatePluginRelease(ctx, adminAccessToken, pluginName, "")
}

func (c *Client) UpdatePluginRelease(ctx context.Context, adminAccessToken, pluginName, version string) error {
	var apiURL string
	switch {
	case pluginName == "" && version == "":
		apiURL = "plugins"
	case pluginName == "" && version != "":
		return fmt.Errorf("plugin name is required when version is specified")
	case pluginName != "" && version == "":
		apiURL = getPluginURL(pluginName)
	case pluginName != "" && version != "":
		apiURL = getPluginReleaseURL(pluginName, version)
	}
	resp, err := c.sendRequest(ctx, http.MethodPut, apiURL, nil, setAuth(adminAccessToken))
	if err != nil {
		return err
	}
	var updateResponse map[string]bool
	err = c.decodeResponse(resp, &updateResponse)
	if err != nil {
		return err
	}
	if !updateResponse["ok"] {
		return fmt.Errorf("update plugin %s@%s failed: reason unknown", pluginName, version)
	}
	return nil
}
