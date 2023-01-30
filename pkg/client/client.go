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
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) sendRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	url, err := url.JoinPath(c.registryURL, endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
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

func (c *Client) GetPlugin(ctx context.Context, plugin string) (*registry.Plugin, error) {
	resp, err := c.sendRequest(ctx, http.MethodGet, fmt.Sprintf("plugins/%s", plugin), nil)
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

func (c *Client) GetPluginRelease(ctx context.Context, plugin, version string) (*registry.PluginRelease, error) {
	resp, err := c.sendRequest(ctx, http.MethodGet, fmt.Sprintf("plugins/%s/versions/%s", plugin, version), nil)
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
