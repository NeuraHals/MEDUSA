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

	// Kafka output
	KafkaBrokers       []string
	KafkaOutputTopic   string
	KafkaDLQTopic      string
	KafkaInputTopic    string
	KafkaConsumerGroup string
	KafkaWorkerCount   int

	// HOA gRPC endpoint for forwarding approval decisions
	HOAGRPCEndpoint string

	// Push — APNs
	APNsKeyPath  string
	APNsTeamID   string
	APNsBundleID string

	// Push — FCM
	FCMServerKey string

	// Twilio SMS fallback
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
	TwilioBaseURL    string

	// Offline queue TTL in seconds
	OfflineQueueTTLSecs int

	// Mobile session TTL
	SessionTTLSecs int

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
	viper.SetDefault("http.addr", ":8087")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.output_topic", "clinical.orchestration.approval.v1")
	viper.SetDefault("kafka.dlq_topic", "system.dlq.v1")
	viper.SetDefault("kafka.input_topic", "clinical.orchestration.execution.v1")
	viper.SetDefault("kafka.consumer_group", "mobile-group")
	viper.SetDefault("kafka.worker_count", 4)
	viper.SetDefault("hoa.grpc_endpoint", "oversight-agent:50051")
	viper.SetDefault("twilio.base_url", "https://api.twilio.com/2010-04-01")
	viper.SetDefault("offline_queue_ttl_secs", 300)
	viper.SetDefault("session_ttl_secs", 3600)
	viper.SetDefault("degraded_mode", false)
	viper.SetDefault("schema.version", "1.0.0")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found: %v", err)
	}

	rd, _ := time.ParseDuration(viper.GetString("http.read_timeout"))
	if rd == 0 { rd = 5 * time.Second }
	wd, _ := time.ParseDuration(viper.GetString("http.write_timeout"))
	if wd == 0 { wd = 10 * time.Second }

	return &Config{
		Env:                 viper.GetString("env"),
		HTTPAddr:            viper.GetString("http.addr"),
		ReadTimeout:         rd,
		WriteTimeout:        wd,
		RedisURL:            viper.GetString("redis.url"),
		OTelEndpoint:        viper.GetString("otel.endpoint"),
		SchemaVersion:       viper.GetString("schema.version"),
		KafkaBrokers:        viper.GetStringSlice("kafka.brokers"),
		KafkaOutputTopic:    viper.GetString("kafka.output_topic"),
		KafkaDLQTopic:       viper.GetString("kafka.dlq_topic"),
		KafkaInputTopic:     viper.GetString("kafka.input_topic"),
		KafkaConsumerGroup:  viper.GetString("kafka.consumer_group"),
		KafkaWorkerCount:    viper.GetInt("kafka.worker_count"),
		HOAGRPCEndpoint:     viper.GetString("hoa.grpc_endpoint"),
		APNsKeyPath:         viper.GetString("apns.key_path"),
		APNsTeamID:          viper.GetString("apns.team_id"),
		APNsBundleID:        viper.GetString("apns.bundle_id"),
		FCMServerKey:        viper.GetString("fcm.server_key"),
		TwilioAccountSID:    viper.GetString("twilio.account_sid"),
		TwilioAuthToken:     viper.GetString("twilio.auth_token"),
		TwilioFromNumber:    viper.GetString("twilio.from_number"),
		TwilioBaseURL:       viper.GetString("twilio.base_url"),
		OfflineQueueTTLSecs: viper.GetInt("offline_queue_ttl_secs"),
		SessionTTLSecs:      viper.GetInt("session_ttl_secs"),
		DegradedMode:        viper.GetBool("degraded_mode"),
	}
}
