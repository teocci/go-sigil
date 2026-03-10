package security

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go-sigil/internal/constants"
)

const binaryProbeBytes = 8192

// IsBinary reports whether the file at path is binary.
// Detection reads up to 8 KB and checks whether the null-byte ratio exceeds
// constants.DefaultBinaryNullThresh (0.1%).
// Empty files are not considered binary.
func IsBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open for binary check: %w", err)
	}
	defer f.Close()

	buf := make([]byte, binaryProbeBytes)
	n, err := io.ReadAtLeast(f, buf, 1)
	if err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			// File is shorter than binaryProbeBytes — use what we got.
			if n == 0 {
				return false, nil // empty file
			}
		} else {
			return false, fmt.Errorf("read for binary check: %w", err)
		}
	}
	buf = buf[:n]

	nulls := bytes.Count(buf, []byte{0})
	return float64(nulls)/float64(n) > constants.DefaultBinaryNullThresh, nil
}
