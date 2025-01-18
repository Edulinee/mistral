package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	PgHost        string `mapstructure:"PG_HOST"`
	PgPort        string `mapstructure:"PG_PORT"`
	PgUser        string `mapstructure:"PG_USER"`
	PgPassword    string `mapstructure:"PG_PASSWORD"`
	PgName        string `mapstructure:"PG_NAME"`
	MistralApiKey string `mapstructure:"MISTRAL_API_KEY"`
	ModelName     string `mapstructure:"MODEL_NAME"`
}

func NewConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config unmarshal error: %w", err)
	}

	return &cfg, nil
}
