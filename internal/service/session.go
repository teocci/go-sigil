// Package service implements business logic for the Sigil indexing pipeline.
package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewSessionID generates a unique session identifier.
// Format: "s_" + 8 random hex chars (10 chars total).
func NewSessionID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("generate session ID: %v", err))
	}
	return "s_" + hex.EncodeToString(b[:])
}
