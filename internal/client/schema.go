package client

import (
	"encoding/json"
	"fmt"
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

	// The API returns the nodes array directly in the data field
	var nodes []NodeSchema
	if err := json.Unmarshal(data, &nodes); err != nil {
		// Fall back to wrapped format
		var result SchemaNodesResponse
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parsing schema nodes: %w", err)
		}
		return &result, nil
	}
	return &SchemaNodesResponse{Nodes: nodes}, nil
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
func (c *Client) EvaluatePipelineJSON(pipeline map[string]any, base64 bool) (*EvaluateJSONResponse, error) {
	body, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("encoding pipeline: %w", err)
	}

	resp, err := c.post("/api/v1/pipeline/evaluate", body)
	if err != nil {
		return nil, err
	}

	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}

	var result EvaluateJSONResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing evaluation result: %w", err)
	}
	return &result, nil
}
