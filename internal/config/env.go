package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func GetEnv(key string, defaultValue ...string) string {
	fileKey := key + "_FILE"
	if filePath := os.Getenv(fileKey); filePath != "" {
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			return defaultValueOrDefault(defaultValue)
		}
		absFile = filepath.Clean(absFile)
		if strings.Contains(filePath, "..") {
			return defaultValueOrDefault(defaultValue)
		}
		data, err := os.ReadFile(absFile)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	if val := os.Getenv(key); val != "" {
		return val
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func GetEnvRequired(key string) string {
	val := GetEnv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return val
}

func GetEnvInt(key string, defaultValue int) int {
	val := GetEnv(key)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return n
}

func GetEnvInt64(key string, defaultValue int64) int64 {
	val := GetEnv(key)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultValue
	}
	return n
}

func GetEnvBool(key string, defaultValue bool) bool {
	val := GetEnv(key)
	if val == "" {
		return defaultValue
	}
	switch strings.ToLower(val) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return defaultValue
}

func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	val := GetEnv(key)
	if val == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultValue
	}
	return d
}

func GetEnvFloat64(key string, defaultValue float64) float64 {
	val := GetEnv(key)
	if val == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

func defaultValueOrDefault(defaultValue []string) string {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}
