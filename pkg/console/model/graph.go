package model

import (
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

// GraphNode represents a node in the graph for AntV G6
type GraphNode struct {
	ID    string      `json:"id"`
	Label string      `json:"label"`
	Data  interface{} `json:"data,omitempty"` // Additional data for the node
}

// GraphEdge represents an edge in the graph for AntV G6
type GraphEdge struct {
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Data   map[string]interface{} `json:"data,omitempty"` // Additional data for the edge
}

// GraphData represents the complete graph structure for AntV G6
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// CrossNode represents a node in the cross-linked list structure
type CrossNode struct {
	Instance *meshresource.InstanceResource
	Next     *CrossNode // pointer to next node in the same row
	Down     *CrossNode // pointer to next node in the same column
}

// CrossLinkedListGraph represents the cross-linked list structure as a directed graph
type CrossLinkedListGraph struct {
	Head *CrossNode
	Rows int // number of rows
	Cols int // number of columns
}

// ServiceGraphReq represents the request parameters for fetching the service graph
type ServiceGraphReq struct {
	ServiceName string `json:"serviceName"  form:"serviceName"`
	Mesh        string `json:"mesh" form:"mesh" binding:"required"`
}
