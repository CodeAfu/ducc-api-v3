package envutils

import "os"

func GetString(key string) (string, error) {
	if val := os.Getenv(key); val != "" {
		return val, nil
	}

	return "", os.ErrNotExist
}
