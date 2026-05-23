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

	// Kafka
	KafkaBrokers       []string
	KafkaInputTopics   []string // subscribes to all orchestration event topics
	KafkaDLQTopic      string
	KafkaConsumerGroup string
	KafkaWorkerCount   int

	// S3 / WORM
	S3Bucket          string
	S3Region          string
	S3Prefix          string
	S3ObjectLockMode  string // COMPLIANCE | GOVERNANCE
	// Standard retention in days (HIPAA minimum = 2555 days / 7 years)
	RetentionDays     int
	ExtendedRetentionDays int // 10-year class
	ForensicRetentionDays int // 25-year class

	// Audit chain
	ChainHashAlgorithm string // SHA-256 (fixed)
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("env", "production")
	viper.SetDefault("http.addr", ":8088")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.dlq_topic", "system.dlq.v1")
	viper.SetDefault("kafka.consumer_group", "audit-group")
	viper.SetDefault("kafka.worker_count", 4)
	viper.SetDefault("s3.prefix", "audit/")
	viper.SetDefault("s3.object_lock_mode", "COMPLIANCE")
	viper.SetDefault("s3.retention_days", 2555)        // 7 years
	viper.SetDefault("s3.extended_retention_days", 3650) // 10 years
	viper.SetDefault("s3.forensic_retention_days", 9125) // 25 years
	viper.SetDefault("chain.hash_algorithm", "SHA-256")
	viper.SetDefault("schema.version", "1.0.0")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found: %v", err)
	}

	rd, _ := time.ParseDuration(viper.GetString("http.read_timeout"))
	if rd == 0 { rd = 5 * time.Second }
	wd, _ := time.ParseDuration(viper.GetString("http.write_timeout"))
	if wd == 0 { wd = 10 * time.Second }

	return &Config{
		Env:                   viper.GetString("env"),
		HTTPAddr:              viper.GetString("http.addr"),
		ReadTimeout:           rd,
		WriteTimeout:          wd,
		RedisURL:              viper.GetString("redis.url"),
		OTelEndpoint:          viper.GetString("otel.endpoint"),
		SchemaVersion:         viper.GetString("schema.version"),
		KafkaBrokers:          viper.GetStringSlice("kafka.brokers"),
		KafkaInputTopics:      viper.GetStringSlice("kafka.input_topics"),
		KafkaDLQTopic:         viper.GetString("kafka.dlq_topic"),
		KafkaConsumerGroup:    viper.GetString("kafka.consumer_group"),
		KafkaWorkerCount:      viper.GetInt("kafka.worker_count"),
		S3Bucket:              viper.GetString("s3.bucket"),
		S3Region:              viper.GetString("s3.region"),
		S3Prefix:              viper.GetString("s3.prefix"),
		S3ObjectLockMode:      viper.GetString("s3.object_lock_mode"),
		RetentionDays:         viper.GetInt("s3.retention_days"),
		ExtendedRetentionDays: viper.GetInt("s3.extended_retention_days"),
		ForensicRetentionDays: viper.GetInt("s3.forensic_retention_days"),
		ChainHashAlgorithm:    viper.GetString("chain.hash_algorithm"),
	}
}
