package sign

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/davidahmann/wrkr/core/jcs"
)

func SHA256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func CanonicalJSONSHA256Hex(data []byte) (string, error) {
	canonical, err := jcs.Canonicalize(data)
	if err != nil {
		return "", fmt.Errorf("canonical digest: %w", err)
	}
	return SHA256Hex(canonical), nil
}
