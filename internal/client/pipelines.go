package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ListPipelines returns the user's pipelines with pagination
func (c *Client) ListPipelines(offset, pageSize int) (*PipelineListResponse, error) {
	path := fmt.Sprintf("/api/v1/pipelines?offset=%d&pageSize=%d", offset, pageSize)
	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result PipelineListResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing pipeline list: %w", err)
	}
	return &result, nil
}

// DeletePipeline deletes a pipeline by ID
func (c *Client) DeletePipeline(id string) error {
	path := fmt.Sprintf("/api/v1/pipelines/%s", id)
	resp, err := c.delete(path)
	if err != nil {
		return err
	}

	_, err = decodeJSON(resp)
	return err
}

// DescribePipeline returns the parameter description of a pipeline
func (c *Client) DescribePipeline(id string) (*PipelineDescription, error) {
	path := fmt.Sprintf("/api/v1/pipelines/%s/describe", id)
	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result PipelineDescription
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing pipeline description: %w", err)
	}
	return &result, nil
}

// EvaluatePipeline executes a pipeline with the given parameters
func (c *Client) EvaluatePipeline(id string, params map[string]any, base64 bool) (*PipelineReport, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encoding parameters: %w", err)
	}

	path := fmt.Sprintf("/api/v1/pipelines/%s/evaluate", id)
	if base64 {
		path += "?format=base64"
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401). Run 'inodes configure' to set your API key")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("pipeline not found")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		// Try server's jsonResponse format: {"error": bool, "message": "..."}
		var errResp struct {
			Error   any    `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Message)
		}
		// Try structured error: {"error": "...", "details": ...}
		if s, ok := errResp.Error.(string); ok && s != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, s)
		}
		if len(body) > 0 {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error (%d)", resp.StatusCode)
	}

	// The evaluate endpoint returns the report directly (not wrapped in jsonResponse)
	var report PipelineReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("parsing evaluation result: %w", err)
	}
	return &report, nil
}

// GetPipeline returns the full pipeline including graph data
func (c *Client) GetPipeline(id string) (*PipelineFull, error) {
	path := fmt.Sprintf("/api/v1/pipelines/%s", id)
	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result PipelineFullResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing pipeline: %w", err)
	}
	return &result.PipelineFull, nil
}

// DiffAssessment runs a diff assessment between API and editor mode for a pipeline
func (c *Client) DiffAssessment(id string) (*DiffAssessmentResult, error) {
	path := fmt.Sprintf("/api/v1/pipelines/%s/diff-assessment", id)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result DiffAssessmentResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing diff assessment: %w", err)
	}
	return &result, nil
}

// DownloadFile downloads a file from a URL (relative or absolute) and returns the bytes
func (c *Client) DownloadFile(path string) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, c.ResolveURL(path), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed (%d)", resp.StatusCode)
	}

	data, err := readLimited(resp.Body, 100*1024*1024) // 100MB max
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

func readLimited(r interface{ Read([]byte) (int, error) }, limit int64) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(&limitedReader{r: r, remaining: limit})
	return buf.Bytes(), err
}

type limitedReader struct {
	r         interface{ Read([]byte) (int, error) }
	remaining int64
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	if lr.remaining <= 0 {
		return 0, fmt.Errorf("response too large (>%d bytes)", lr.remaining)
	}
	if int64(len(p)) > lr.remaining {
		p = p[:lr.remaining]
	}
	n, err := lr.r.Read(p)
	lr.remaining -= int64(n)
	return n, err
}

// ErrPipelineNotFound is returned (wrapped) when the server reports 404 for a
// pipeline-scoped request.
var ErrPipelineNotFound = errors.New("pipeline not found")

// ErrVersionConflict is returned (wrapped) when an optimistic-locked update is
// rejected because the stored pipeline version has moved on.
var ErrVersionConflict = errors.New("version conflict")

// UpdatePipeline replaces the graph of an existing pipeline in place via
// PUT /api/v1/pipelines/{id}. Only the graph is replaced; the pipeline's name
// and description are left unchanged. version > 0 enables optimistic locking.
// The server uses the URL id, so any "id" inside the graph is ignored.
func (c *Client) UpdatePipeline(id string, pipeline map[string]any, version int) (*UpdatePipelineResponse, error) {
	body, err := json.Marshal(UpdatePipelineRequest{Pipeline: pipeline, Version: version})
	if err != nil {
		return nil, fmt.Errorf("encoding update request: %w", err)
	}

	resp, err := c.put("/api/v1/pipelines/"+url.PathEscape(id), body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		serverMessage(resp) // drain and close
		return nil, fmt.Errorf("API error (404): %w: %s. Check the ID with 'inodes list'", ErrPipelineNotFound, id)
	case http.StatusConflict:
		msg := serverMessage(resp)
		if msg == "" {
			msg = "pipeline was modified since the expected version"
		}
		return nil, fmt.Errorf("API error (409): %w: %s. Re-export the latest version with 'inodes export %s', re-apply your changes, and try again", ErrVersionConflict, msg, id)
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result UpdatePipelineResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing update result: %w", err)
	}
	return &result, nil
}

// serverMessage reads and closes the response body, returning the "message"
// field of the standard {"error", "message"} envelope if present.
func serverMessage(resp *http.Response) string {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var env struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &env) != nil {
		return ""
	}
	return env.Message
}
