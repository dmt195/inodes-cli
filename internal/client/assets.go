package client

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// UploadEphemeral uploads a file as an ephemeral asset and returns its ID and expiry
func (c *Client) UploadEphemeral(filePath string) (*EphemeralUploadResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/assets/ephemeral", pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result EphemeralUploadResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing upload response: %w", err)
	}
	return &result, nil
}

// DeleteEphemeral deletes an ephemeral asset by ID, freeing a slot against the per-user limit.
func (c *Client) DeleteEphemeral(id string) error {
	path := fmt.Sprintf("/api/v1/assets/ephemeral/%s", id)
	resp, err := c.delete(path)
	if err != nil {
		return err
	}
	_, err = decodeJSON(resp)
	return err
}
