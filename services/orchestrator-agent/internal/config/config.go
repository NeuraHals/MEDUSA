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
	KafkaBrokers     []string
	KafkaInputTopic  string
	KafkaOutputTopic string
	KafkaDLQTopic    string
	KafkaConsumerGroup string
	KafkaWorkerCount int
	RedisURL         string
	OTelEndpoint     string
	SchemaVersion    string
	// HOA gRPC endpoint for approval token validation
	HOAEndpoint      string
	// Maximum execution time before FAILED state is forced
	ExecutionTimeoutSecs int
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("env", "production")
	viper.SetDefault("http.addr", ":8084")
	viper.SetDefault("grpc.addr", ":50052")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.input_topic", "clinical.orchestration.blueprint.v1")
	viper.SetDefault("kafka.output_topic", "clinical.orchestration.execution.v1")
	viper.SetDefault("kafka.dlq_topic", "system.dlq.v1")
	viper.SetDefault("kafka.consumer_group", "orchestrator-group")
	viper.SetDefault("kafka.worker_count", 4)
	viper.SetDefault("schema.version", "1.0.0")
	viper.SetDefault("execution_timeout_secs", 30)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found, using env vars: %v", err)
	}

	readTimeout, _ := time.ParseDuration(viper.GetString("http.read_timeout"))
	if readTimeout == 0 {
		readTimeout = 5 * time.Second
	}
	writeTimeout, _ := time.ParseDuration(viper.GetString("http.write_timeout"))
	if writeTimeout == 0 {
		writeTimeout = 10 * time.Second
	}

	return &Config{
		Env:                  viper.GetString("env"),
		HTTPAddr:             viper.GetString("http.addr"),
		GRPCAddr:             viper.GetString("grpc.addr"),
		ReadTimeout:          readTimeout,
		WriteTimeout:         writeTimeout,
		KafkaBrokers:         viper.GetStringSlice("kafka.brokers"),
		KafkaInputTopic:      viper.GetString("kafka.input_topic"),
		KafkaOutputTopic:     viper.GetString("kafka.output_topic"),
		KafkaDLQTopic:        viper.GetString("kafka.dlq_topic"),
		KafkaConsumerGroup:   viper.GetString("kafka.consumer_group"),
		KafkaWorkerCount:     viper.GetInt("kafka.worker_count"),
		RedisURL:             viper.GetString("redis.url"),
		OTelEndpoint:         viper.GetString("otel.endpoint"),
		SchemaVersion:        viper.GetString("schema.version"),
		HOAEndpoint:          viper.GetString("hoa.grpc_endpoint"),
		ExecutionTimeoutSecs: viper.GetInt("execution_timeout_secs"),
	}
}
