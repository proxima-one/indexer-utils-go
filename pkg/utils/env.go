package utils

import (
	"fmt"
	"os"
)

func GetEnvOrDefault(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func RequireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprint("no env key", key))
	}
	return value
}
