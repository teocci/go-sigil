// Package models defines domain types used across the application.
// This package has no dependencies on other internal packages.
package models

// File represents a tracked source file in the index.
type File struct {
	Path        string `json:"path"`
	BlobSHA     string `json:"blob_sha,omitempty"`     // empty in filesystem mode
	Mtime       string `json:"mtime,omitempty"`        // filesystem mode
	Size        int64  `json:"size,omitempty"`         // filesystem mode
	LastIndexed string `json:"last_indexed"`
}
