package client

import "time"

// JSONResponse mirrors the server's generic JSON response wrapper
type JSONResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Pipeline represents an abridged pipeline from the list endpoint
type Pipeline struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	LastImage   string `json:"last_image,omitempty"`
	IsFavourite bool   `json:"is_favourite"`
	IsLocked    bool   `json:"is_locked"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// PipelineListResponse is the data payload for listing pipelines
type PipelineListResponse struct {
	Pipelines []Pipeline `json:"pipelines"`
	Meta      struct {
		Count       int `json:"count"`
		Offset      int `json:"offset"`
		PageSize    int `json:"pageSize"`
		CurrentPage int `json:"currentPage"`
		TotalPages  int `json:"totalPages"`
	} `json:"meta"`
}

// PipelineDescription describes a pipeline's API parameters
type PipelineDescription struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	EvaluateLink  string               `json:"evaluate_link,omitempty"`
	DescribeLink  string               `json:"describe_link,omitempty"`
	ApiNodes      []ApiValueDescriptor `json:"api_nodes"`
	ApiImageNodes []ApiImageDescriptor `json:"api_image_nodes,omitempty"`
}

// ApiValueDescriptor describes a value parameter
type ApiValueDescriptor struct {
	Key          string `json:"key"`
	DataType     string `json:"data_type"`
	DefaultValue any    `json:"default"`
}

// ApiImageDescriptor describes an image parameter
type ApiImageDescriptor struct {
	Key      string `json:"key"`
	Required bool   `json:"required"`
}

// PipelineReport is the result of evaluating a pipeline
type PipelineReport struct {
	Success             bool            `json:"success"`
	ImageDetails        ImageDetails    `json:"image_details"`
	TotalProcessingTime time.Duration   `json:"total_processing_time"`
	TotalUnitsBillable  int             `json:"total_processing_units"`
	NodesPerformance    []NodePerfEntry `json:"nodes_performance,omitempty"`
}

// ImageDetails holds the result image info
type ImageDetails struct {
	ImageAsBase64 string `json:"image_as_base_64,omitempty"`
	ImageUrl      string `json:"image_url,omitempty"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	ImageHash     string `json:"image_hash,omitempty"`
	Format        string `json:"format,omitempty"`
	Quality       int    `json:"quality,omitempty"`
}

// NodePerfEntry is a single node's performance data
type NodePerfEntry struct {
	NodeID         string        `json:"node_id"`
	NodeType       string        `json:"node_type"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// DiffAssessmentResult is the result of a diff assessment between API and editor mode
type DiffAssessmentResult struct {
	AvgDiff        float64 `json:"avgDiff"`
	MaxDiff        int     `json:"maxDiff"`
	ApiWidth       int     `json:"apiWidth"`
	ApiHeight      int     `json:"apiHeight"`
	EditorWidth    int     `json:"editorWidth"`
	EditorHeight   int     `json:"editorHeight"`
	ScaleFactor    float64 `json:"scaleFactor"`
	PixelsCompared int     `json:"pixelsCompared"`
}

// EphemeralUploadResponse is the data from uploading an ephemeral asset
type EphemeralUploadResponse struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}
