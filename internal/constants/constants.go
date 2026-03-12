// Package constants defines immutable application-wide values.
package constants

import "time"

// Application identity.
const (
	AppName    = "sigil"
	AppVersion = "0.1.3"
)

// Database schema.
const (
	SchemaVersion  = 1
	MinSafeVersion = 1
)

// SymbolKind matches the kind column in the symbols table.
type SymbolKind string

const (
	KindFunction  SymbolKind = "function"
	KindClass     SymbolKind = "class"
	KindMethod    SymbolKind = "method"
	KindInterface SymbolKind = "interface"
	KindType      SymbolKind = "type"
	KindConst     SymbolKind = "const"
	KindVar       SymbolKind = "var"
)

// Confidence represents call edge confidence levels.
type Confidence string

const (
	ConfidenceStatic   Confidence = "static"
	ConfidenceInferred Confidence = "inferred"
	ConfidenceDynamic  Confidence = "dynamic"
)

// EnvVarState represents environment variable state classifications for sigil_env.
type EnvVarState string

const (
	EnvStateSet         EnvVarState = "set"
	EnvStateEmpty       EnvVarState = "empty"
	EnvStatePlaceholder EnvVarState = "placeholder"
	EnvStateUnset       EnvVarState = "unset"
	EnvStateMissing     EnvVarState = "missing"
)

// Default operational limits.
const (
	DefaultMaxIndexFiles     = 500
	DefaultEnrichBatchSize   = 4
	DefaultMaxFileSize       = 2 << 20 // 2 MB
	DefaultBinaryNullThresh  = 0.001
	DefaultContextLinesMax   = 50
	DefaultSearchLimit       = 10
	DefaultBusyTimeoutMs     = 5000
	DefaultEnrichTimeout     = 60 * time.Second
	DefaultBackwardScanChunk = 512
)
