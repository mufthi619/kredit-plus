package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Logger    LoggerConfig    `mapstructure:"logger"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Port        int    `mapstructure:"port"`
}

type MySQLConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
	Database     string        `mapstructure:"database"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
	Debug        bool          `mapstructure:"debug"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LoggerConfig struct {
	Level       string `mapstructure:"level"`
	Environment string `mapstructure:"environment"`
}

type TelemetryConfig struct {
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`
	Environment    string `mapstructure:"environment"`
	OTLPEndpoint   string `mapstructure:"otlp_endpoint"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
