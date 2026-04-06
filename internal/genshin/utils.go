package genshin

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
)

func getKeyFromBearerToken(token, key string) (string, error) {
	parts := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var raw map[string]any
	json.Unmarshal(payload, &raw)
	return raw[key].(string), nil
}

func isUserAllowedToAccess(token string, allowedEmails []string) (bool, error) {
	email, err := getKeyFromBearerToken(token, "email")
	if err != nil {
		return false, err
	}
	email = strings.ToLower(email)
	lowered := make([]string, len(allowedEmails))
	for i, e := range allowedEmails {
		lowered[i] = strings.ToLower(e)
	}
	return slices.Contains(lowered, email), nil
}
