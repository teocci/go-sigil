// Package constants — security-related patterns and exclusion lists.
package constants

// BuiltinRedactedPatterns are file patterns always treated as redacted
// (keys visible, values nulled). Cannot be removed via config.
var BuiltinRedactedPatterns = []string{
	".env",
	"*.env.*",
	"*.pem",
	"*.key",
	"*.p12",
	"id_rsa*",
	"credentials.*",
	".netrc",
}

// BuiltinExclusions are directories and file patterns always skipped during indexing.
var BuiltinExclusions = []string{
	"node_modules/",
	"vendor/",
	".git/",
	"__pycache__/",
	"target/",
	"dist/",
	"build/",
	"*.min.js",
	"*.min.css",
	"*.lock",
	"*.map",
	"*.wasm",
}

// SecretValuePatterns are regex patterns applied to constant values to detect secrets.
// Matched values are redacted in the symbol summary and have byte offsets nulled.
var SecretValuePatterns = []string{
	`(?i)api[_-]?key\s*[:=]\s*["']?[A-Za-z0-9_\-]{20,}`,
	`(?i)secret\s*[:=]\s*["']?[A-Za-z0-9_\-]{20,}`,
	`(?i)password\s*[:=]\s*["']?[^\s"']{8,}`,
	`(?i)token\s*[:=]\s*["']?[A-Za-z0-9_\-]{20,}`,
	`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`,
	`(?i)(postgres|mysql|mongodb|redis)://[^\s]+`,
}

// PlaceholderPatterns detect placeholder values in environment variables.
var PlaceholderPatterns = []string{
	`^x+$`,
	`YOUR_.*`,
	`<[^>]+>`,
	`TODO`,
	`changeme`,
	`^dummy`,
	`^fake`,
	`^example`,
	`^0+$`,
	`^\*+$`,
	`REPLACE_.*`,
}
