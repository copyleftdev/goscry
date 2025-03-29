package mcp

import (
	"time"
)

const MCPVersion = "2025-03-26"

type Message struct {
	MCPVersion string  `json:"mcp_version"`
	Context    Context `json:"context"`
	RequestID  string  `json:"request_id,omitempty"`
	TaskID     string  `json:"task_id,omitempty"`
}

type Context struct {
	Metadata Metadata `json:"metadata"`
	Actors   []Actor  `json:"actors,omitempty"`
	Content  Content  `json:"content"`
	ParentID string   `json:"parent_id,omitempty"`
	Schema   string   `json:"schema,omitempty"`
}

type Metadata struct {
	SourceURI string                 `json:"source_uri,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

type Actor struct {
	ID     string                 `json:"id"`
	Role   string                 `json:"role"`
	Custom map[string]interface{} `json:"custom,omitempty"`
}

type Content struct {
	MIMEType string                 `json:"mime_type"`
	Data     interface{}            `json:"data"`
	Encoding string                 `json:"encoding,omitempty"` // e.g., "base64"
	Custom   map[string]interface{} `json:"custom,omitempty"`
}

func NewBaseMessage(taskID string) Message {
	return Message{
		MCPVersion: MCPVersion,
		TaskID:     taskID,
		Context: Context{
			Metadata: Metadata{
				Timestamp: time.Now().UTC(),
			},
			Actors: []Actor{
				{ID: "goscry-agent", Role: "browser_automation_tool"}, // Default actor
			},
		},
	}
}
