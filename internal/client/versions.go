package client

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
)

// PipelineVersion is one saved version of a pipeline's graph.
type PipelineVersion struct {
	Version      int    `json:"version"`
	SavedBy      int    `json:"saved_by"`
	SavedByEmail string `json:"saved_by_email"`
	CreatedAt    string `json:"created_at"`
}

// ListVersions returns a pipeline's saved versions, newest first.
func (c *Client) ListVersions(pipelineID string) ([]PipelineVersion, error) {
	resp, err := c.get("/api/v1/pipelines/" + url.PathEscape(pipelineID) + "/versions")
	if err != nil {
		return nil, err
	}
	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}
	var result struct {
		Versions []PipelineVersion `json:"versions"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing versions: %w", err)
	}
	return result.Versions, nil
}

// RestoreVersion makes an earlier version current and returns the new
// version number the restore was saved as.
func (c *Client) RestoreVersion(pipelineID string, version int) (int, error) {
	resp, err := c.post(fmt.Sprintf("/api/v1/pipelines/%s/versions/%d/restore", url.PathEscape(pipelineID), version), nil)
	if err != nil {
		return 0, err
	}
	data, err := decodeJSON(resp)
	if err != nil {
		return 0, err
	}
	var result struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("parsing restore result: %w", err)
	}
	return result.Version, nil
}

// DownloadAsset downloads an asset library image, returning its bytes,
// content type and original filename (if the server sent one).
func (c *Client) DownloadAsset(assetID string) ([]byte, string, string, error) {
	resp, err := c.get("/api/v1/assets/" + url.PathEscape(assetID) + "/file")
	if err != nil {
		return nil, "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		_, err := decodeJSON(resp) // closes the body and reads the server's message
		if err == nil {
			err = fmt.Errorf("download failed (%d)", resp.StatusCode)
		}
		return nil, "", "", err
	}
	defer resp.Body.Close()

	data, err := readLimited(resp.Body, 100*1024*1024) // 100MB max
	if err != nil {
		return nil, "", "", err
	}
	var filename string
	if _, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition")); err == nil {
		filename = params["filename"]
	}
	return data, resp.Header.Get("Content-Type"), filename, nil
}
