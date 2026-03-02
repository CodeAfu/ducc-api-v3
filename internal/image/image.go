package image

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func GenerateImageHash(base64Str string) string {
	hash := sha256.Sum256([]byte(base64Str))
	return hex.EncodeToString(hash[:])
}

func DecodeImage(base64Str string) ([]byte, error) {
	imageData, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}
