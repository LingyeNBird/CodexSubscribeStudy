// Package protocol contains the immutable, versioned research design.
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

//go:embed method-v2.json
var MethodV2 []byte

func DigestV2() string { return fmt.Sprintf("%x", sha256.Sum256(bytes.TrimSpace(MethodV2))) }
