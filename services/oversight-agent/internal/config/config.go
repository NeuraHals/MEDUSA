package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env              string
	HTTPAddr         string
	GRPCAddr         string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	RedisURL         string
	KafkaBrokers     []string
	KafkaOutputTopic string
	KafkaDLQTopic    string
	OTelEndpoint     string
	SchemaVersion    string
	// Glass Break timer — 60 seconds per v1.0 architecture contract
	ApprovalTimeoutSecs int
	// Degraded mode disables biometric validation (for offline operation)
	DegradedMode bool
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("env", "production")
	viper.SetDefault("http.addr", ":8085")
	viper.SetDefault("grpc.addr", ":50051")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.output_topic", "clinical.orchestration.approval.v1")
	viper.SetDefault("kafka.dlq_topic", "system.dlq.v1")
	viper.SetDefault("approval_timeout_secs", 60)
	viper.SetDefault("degraded_mode", false)
	viper.SetDefault("schema.version", "1.0.0")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found, using env vars: %v", err)
	}

	readTimeout, _ := time.ParseDuration(viper.GetString("http.read_timeout"))
	if readTimeout == 0 { readTimeout = 5 * time.Second }
	writeTimeout, _ := time.ParseDuration(viper.GetString("http.write_timeout"))
	if writeTimeout == 0 { writeTimeout = 10 * time.Second }

	return &Config{
		Env:                 viper.GetString("env"),
		HTTPAddr:            viper.GetString("http.addr"),
		GRPCAddr:            viper.GetString("grpc.addr"),
		ReadTimeout:         readTimeout,
		WriteTimeout:        writeTimeout,
		RedisURL:            viper.GetString("redis.url"),
		KafkaBrokers:        viper.GetStringSlice("kafka.brokers"),
		KafkaOutputTopic:    viper.GetString("kafka.output_topic"),
		KafkaDLQTopic:       viper.GetString("kafka.dlq_topic"),
		OTelEndpoint:        viper.GetString("otel.endpoint"),
		SchemaVersion:       viper.GetString("schema.version"),
		ApprovalTimeoutSecs: viper.GetInt("approval_timeout_secs"),
		DegradedMode:        viper.GetBool("degraded_mode"),
	}
}
