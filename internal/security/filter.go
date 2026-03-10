// Package security implements Sigil's three-tier file classification model
// and secret value detection.
//
// Every file processed by the indexing pipeline passes through the security
// filter before parsing:
//
//	TierIgnored   — fully invisible to all Sigil tools
//	TierRedacted  — symbol keys visible; byte offsets and values nulled
//	TierNormal    — full symbol extraction
package security

// Tier is the security classification assigned to a file.
type Tier int

const (
	// TierNormal allows full symbol extraction.
	TierNormal Tier = iota
	// TierRedacted exposes symbol names but nulls byte offsets and values.
	// Applies to .env, *.pem, *.key, and similar files.
	TierRedacted
	// TierIgnored makes the file completely invisible to all Sigil tools.
	// Applies to patterns in config extra_ignore_filenames.
	TierIgnored
)

// Filter classifies files and detects secret values in symbol constants.
// Consumed by the service layer; implemented in classify.go.
type Filter interface {
	// ClassifyFile returns the security tier for the given file path.
	ClassifyFile(path string) Tier
	// IsSecretValue reports whether value matches a known secret pattern.
	IsSecretValue(value string) bool
}
