package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetSchemaNodes returns all available node types and their parameters
func (c *Client) GetSchemaNodes() (*SchemaNodesResponse, error) {
	resp, err := c.get("/api/v1/schema/nodes")
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result SchemaNodesResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing schema nodes: %w", err)
	}
	return &result, nil
}

// ValidatePipeline validates a pipeline definition without executing it
func (c *Client) ValidatePipeline(pipeline map[string]any) (*ValidateResponse, error) {
	body, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("encoding pipeline: %w", err)
	}

	resp, err := c.post("/api/v1/pipeline/validate", body)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result ValidateResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing validation result: %w", err)
	}
	return &result, nil
}

// EstimatePipelineCost estimates the cost of executing a pipeline
func (c *Client) EstimatePipelineCost(pipeline map[string]any) (*EstimateCostResponse, error) {
	body, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("encoding pipeline: %w", err)
	}

	resp, err := c.post("/api/v1/pipeline/estimate-cost", body)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result EstimateCostResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing cost estimate: %w", err)
	}
	return &result, nil
}

// EvaluatePipelineJSON executes a pipeline defined inline as JSON
func (c *Client) EvaluatePipelineJSON(pipeline map[string]any, base64 bool) (*PipelineReport, error) {
	body, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("encoding pipeline: %w", err)
	}

	path := "/api/v1/pipeline/evaluate"
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
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error   any    `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Message)
		}
		if s, ok := errResp.Error.(string); ok && s != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, s)
		}
		if len(body) > 0 {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error (%d)", resp.StatusCode)
	}

	var report PipelineReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("parsing evaluation result: %w", err)
	}
	return &report, nil
}
