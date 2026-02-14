package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the Image Nodes API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a new API client
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// BaseURL returns the configured base URL
func (c *Client) BaseURL() string {
	return c.baseURL
}

// ResolveURL returns an absolute URL. If path is already absolute, it is
// returned as-is; otherwise baseURL is prepended.
func (c *Client) ResolveURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + path
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// decodeJSON reads the response body and decodes the JSON response.
// It returns the raw Data field for further unmarshaling.
func decodeJSON(resp *http.Response) (json.RawMessage, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401). Run 'inodes configure' to set your API key")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found (404)")
	}

	if resp.StatusCode >= 400 {
		// Try to extract error message from JSON
		var errResp struct {
			Error   bool   `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse the wrapper to extract Data field
	var wrapper struct {
		Error   bool            `json:"error"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		// Not a standard wrapper — return raw body as data
		return body, nil
	}

	if wrapper.Error {
		return nil, fmt.Errorf("API error: %s", wrapper.Message)
	}

	if wrapper.Data != nil {
		return wrapper.Data, nil
	}

	// No data field — return full body
	return body, nil
}
