package image

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
)

func CheckValidImage(data []byte) bool {
	mime := http.DetectContentType(data)
	return strings.HasPrefix(mime, "image/")
}

func GenerateImageHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func DecodeImage(base64Str string) ([]byte, error) {
	imageData, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}
