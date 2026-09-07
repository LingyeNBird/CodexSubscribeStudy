// Package protocol contains the immutable research design.
package protocol

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
)

//go:embed method.json
var Method []byte

func Digest() string { return fmt.Sprintf("%x", sha256.Sum256(bytes.TrimSpace(Method))) }
