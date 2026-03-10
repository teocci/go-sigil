package models

// Symbol is the atomic unit of Sigil — a function, class, method,
// interface, constant, type, or variable extracted at index time.
type Symbol struct {
	ID                 string `json:"id"`   // SHA256(qualified_name)[:8]
	Kind               string `json:"kind"` // function|class|method|interface|type|const|var
	Name               string `json:"name"`
	QualifiedName      string `json:"qualified_name"` // full repo-relative path + symbol
	Language           string `json:"language"`
	File               string `json:"file"`
	PackageRoot        string `json:"package_root,omitempty"`
	ByteStart          *int   `json:"byte_start,omitempty"` // nil if redacted
	ByteEnd            *int   `json:"byte_end,omitempty"`
	LineStart          int    `json:"line_start"`
	LineEnd            int    `json:"line_end"`
	Signature          string `json:"signature,omitempty"`
	Summary            string `json:"summary,omitempty"`
	Tags               string `json:"tags,omitempty"` // JSON array
	ParentID           string `json:"parent_id,omitempty"`
	Children           string `json:"children,omitempty"` // JSON array of symbol IDs
	Depth              int    `json:"depth"`
	Imports            string `json:"imports,omitempty"` // JSON array of type names
	ContentHash        string `json:"content_hash"`
	PossibleUnresolved bool   `json:"possible_unresolved,omitempty"`
	Untracked          bool   `json:"untracked,omitempty"`
	IndexedAt          string `json:"indexed_at"`
}
