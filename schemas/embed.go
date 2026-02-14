package schemas

import "embed"

// V1FS stores the shipped v1 schemas and OpenAPI baseline for runtime validation.
//
//go:embed v1/*/*.json
var V1FS embed.FS
