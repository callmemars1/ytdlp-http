package configurations

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	S3     S3Config     `mapstructure:"s3"`
	Auth   AuthConfig   `mapstructure:"auth"`
}

type ServerConfig struct {
	Addr string `mapstructure:"addr"`
}

type S3Config struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	Endpoint        string `mapstructure:"endpoint"`
}

type AuthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	APIKey  string `mapstructure:"api_key"`
}

func NewConfig() (*Config, error) {
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("auth.enabled", false)

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	if config.Auth.Enabled && config.Auth.APIKey == "" {
		log.Fatal("AUTH_API_KEY is required when auth is enabled")
	}

	if config.S3.AccessKeyID == "" || config.S3.SecretAccessKey == "" {
		log.Fatal("S3 credentials are required")
	}

	return &config, nil
}
