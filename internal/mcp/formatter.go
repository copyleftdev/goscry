package mcp

import (
	"encoding/json"

	"golang.org/x/net/html" // Used only if we implement advanced simplification
)

func marshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func FormatStatus(taskID, statusMsg string, sourceURI string) ([]byte, error) {
	msg := NewBaseMessage(taskID)
	msg.Context.Metadata.SourceURI = sourceURI
	msg.Context.Content = Content{
		MIMEType: "text/plain",
		Data:     statusMsg,
	}
	return marshalMessage(msg)
}

func FormatError(taskID string, err error, sourceURI string) ([]byte, error) {
	msg := NewBaseMessage(taskID)
	msg.Context.Metadata.SourceURI = sourceURI
	errorData := map[string]string{
		"error": err.Error(),
	}
	msg.Context.Content = Content{
		MIMEType: "application/json",
		Data:     errorData,
	}
	// Optionally add custom metadata about the error context
	// msg.Context.Metadata.Custom = map[string]interface{}{ ... }
	return marshalMessage(msg)
}

func FormatDOMContent(taskID string, domData interface{}, mimeType string, sourceURI string, encoding string) ([]byte, error) {
	msg := NewBaseMessage(taskID)
	msg.Context.Metadata.SourceURI = sourceURI
	msg.Context.Content = Content{
		MIMEType: mimeType,
		Data:     domData,
		Encoding: encoding, // e.g., "base64" for screenshots
	}
	return marshalMessage(msg)
}

func Format2FARequest(taskID string, promptDetails string, sourceURI string) ([]byte, error) {
	msg := NewBaseMessage(taskID)
	msg.Context.Metadata.SourceURI = sourceURI
	msg.Context.Metadata.Custom = map[string]interface{}{
		"interaction_required": "2fa",
	}
	msg.Context.Content = Content{
		MIMEType: "text/plain",
		Data:     "Two-factor authentication code required: " + promptDetails,
	}
	return marshalMessage(msg)
}

// Placeholder for potential future advanced simplification
func simplifyHTMLNode(node *html.Node) interface{} {
	// This would be a complex function traversing the node tree
	// and building a simplified representation (e.g., map or struct).
	// For now, just return a placeholder description.
	return "Simplified DOM representation logic goes here."
}
