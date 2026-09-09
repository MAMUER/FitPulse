package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

func LoadViper(appName string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(appName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath(filepath.Join("/etc", appName))

	v.SetEnvPrefix(appName)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return v, nil
}

func MustLoadViper(appName string) *viper.Viper {
	v, err := LoadViper(appName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return v
}

func GetString(v *viper.Viper, key, defaultVal string) string {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetString(key)
}

func GetInt(v *viper.Viper, key string, defaultVal int) int {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetInt(key)
}

func GetBool(v *viper.Viper, key string, defaultVal bool) bool {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetBool(key)
}

func GetDuration(v *viper.Viper, key string, defaultVal string) time.Duration {
	val := GetString(v, key, defaultVal)
	d, err := time.ParseDuration(val)
	if err != nil {
		return time.Duration(0)
	}
	return d
}

func GetFloat64(v *viper.Viper, key string, defaultVal float64) float64 {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetFloat64(key)
}

func GetEnvViper(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func GetEnvRequiredViper(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return val
}
