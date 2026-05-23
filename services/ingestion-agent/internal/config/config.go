package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the Signal Ingestion Agent.
type Config struct {
	Env           string
	HTTPAddr      string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	KafkaBrokers  []string
	KafkaTopic    string
	OTelEndpoint  string
	SchemaVersion string
}

// Load reads configuration from config.yaml and environment variable overrides.
// Safe production defaults are applied for all timeout values.
func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Safe production defaults
	viper.SetDefault("env", "production")
	viper.SetDefault("http.addr", ":8080")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.topic", "external.telemetry.v1")
	viper.SetDefault("schema.version", "1.0.0")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found, using env vars: %v", err)
	}

	readTimeout, err := time.ParseDuration(viper.GetString("http.read_timeout"))
	if err != nil {
		readTimeout = 5 * time.Second
	}
	writeTimeout, err := time.ParseDuration(viper.GetString("http.write_timeout"))
	if err != nil {
		writeTimeout = 10 * time.Second
	}

	return &Config{
		Env:           viper.GetString("env"),
		HTTPAddr:      viper.GetString("http.addr"),
		ReadTimeout:   readTimeout,
		WriteTimeout:  writeTimeout,
		KafkaBrokers:  viper.GetStringSlice("kafka.brokers"),
		KafkaTopic:    viper.GetString("kafka.topic"),
		OTelEndpoint:  viper.GetString("otel.endpoint"),
		SchemaVersion: viper.GetString("schema.version"),
	}
}
