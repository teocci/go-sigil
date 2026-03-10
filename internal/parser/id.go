package parser

import (
	"crypto/sha256"
	"fmt"
)

// SymbolID returns the stable 8-char hex ID for a symbol: SHA256(qualifiedName)[:8].
func SymbolID(qualifiedName string) string {
	sum := sha256.Sum256([]byte(qualifiedName))
	return fmt.Sprintf("%x", sum[:4])
}
