package jcs

import (
	"fmt"

	jsoncanonicalizer "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// Canonicalize normalizes arbitrary JSON bytes into RFC 8785 canonical form.
func Canonicalize(input []byte) ([]byte, error) {
	out, err := jsoncanonicalizer.Transform(input)
	if err != nil {
		return nil, fmt.Errorf("canonicalize json: %w", err)
	}
	return out, nil
}
