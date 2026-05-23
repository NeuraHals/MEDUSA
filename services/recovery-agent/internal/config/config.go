package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env          string
	HTTPAddr     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	RedisURL      string
	OTelEndpoint  string
	SchemaVersion string

	KafkaBrokers       []string
	KafkaInputTopic    string
	KafkaOutputTopic   string
	KafkaDLQTopic      string
	KafkaConsumerGroup string
	KafkaWorkerCount   int

	// Rollback execution
	MaxRetries          int
	UndoTimeoutSecs     int
	DegradedMode        bool
	// CB settings
	CBFailureThreshold int
	CBRecoverySecs     int
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("env", "production")
	viper.SetDefault("http.addr", ":8089")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.input_topic", "clinical.orchestration.rollback.v1")
	viper.SetDefault("kafka.output_topic", "clinical.orchestration.recovery.v1")
	viper.SetDefault("kafka.dlq_topic", "system.dlq.v1")
	viper.SetDefault("kafka.consumer_group", "recovery-group")
	viper.SetDefault("kafka.worker_count", 4)
	viper.SetDefault("schema.version", "1.0.0")
	viper.SetDefault("max_retries", 3)
	viper.SetDefault("undo_timeout_secs", 15)
	viper.SetDefault("degraded_mode", false)
	viper.SetDefault("cb.failure_threshold", 5)
	viper.SetDefault("cb.recovery_secs", 30)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found: %v", err)
	}

	rd, _ := time.ParseDuration(viper.GetString("http.read_timeout"))
	if rd == 0 { rd = 5 * time.Second }
	wd, _ := time.ParseDuration(viper.GetString("http.write_timeout"))
	if wd == 0 { wd = 10 * time.Second }

	return &Config{
		Env:                viper.GetString("env"),
		HTTPAddr:           viper.GetString("http.addr"),
		ReadTimeout:        rd,
		WriteTimeout:       wd,
		RedisURL:           viper.GetString("redis.url"),
		OTelEndpoint:       viper.GetString("otel.endpoint"),
		SchemaVersion:      viper.GetString("schema.version"),
		KafkaBrokers:       viper.GetStringSlice("kafka.brokers"),
		KafkaInputTopic:    viper.GetString("kafka.input_topic"),
		KafkaOutputTopic:   viper.GetString("kafka.output_topic"),
		KafkaDLQTopic:      viper.GetString("kafka.dlq_topic"),
		KafkaConsumerGroup: viper.GetString("kafka.consumer_group"),
		KafkaWorkerCount:   viper.GetInt("kafka.worker_count"),
		MaxRetries:         viper.GetInt("max_retries"),
		UndoTimeoutSecs:    viper.GetInt("undo_timeout_secs"),
		DegradedMode:       viper.GetBool("degraded_mode"),
		CBFailureThreshold: viper.GetInt("cb.failure_threshold"),
		CBRecoverySecs:     viper.GetInt("cb.recovery_secs"),
	}
}
