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

	// PagerDuty
	PagerDutyAPIKey    string
	PagerDutyBaseURL   string

	// Twilio
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
	TwilioBaseURL    string

	// Push
	APNsKeyPath    string
	APNsTeamID     string
	APNsBundleID   string
	FCMServerKey   string

	// Circuit breaker
	CBFailureThreshold int
	CBRecoverySeconds  int

	// Retry
	MaxRetries int

	// Degraded mode: if true, falls back to SMS-only
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
	viper.SetDefault("http.addr", ":8086")
	viper.SetDefault("http.read_timeout", "5s")
	viper.SetDefault("http.write_timeout", "10s")
	viper.SetDefault("kafka.input_topic", "clinical.orchestration.execution.v1")
	viper.SetDefault("kafka.output_topic", "clinical.orchestration.notification.v1")
	viper.SetDefault("kafka.dlq_topic", "system.dlq.v1")
	viper.SetDefault("kafka.consumer_group", "messaging-group")
	viper.SetDefault("kafka.worker_count", 8)
	viper.SetDefault("schema.version", "1.0.0")
	viper.SetDefault("pagerduty.base_url", "https://events.pagerduty.com/v2/enqueue")
	viper.SetDefault("twilio.base_url", "https://api.twilio.com/2010-04-01")
	viper.SetDefault("cb.failure_threshold", 5)
	viper.SetDefault("cb.recovery_seconds", 30)
	viper.SetDefault("max_retries", 3)
	viper.SetDefault("degraded_mode", false)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: config file not found, using env vars: %v", err)
	}

	readTimeout, _ := time.ParseDuration(viper.GetString("http.read_timeout"))
	if readTimeout == 0 { readTimeout = 5 * time.Second }
	writeTimeout, _ := time.ParseDuration(viper.GetString("http.write_timeout"))
	if writeTimeout == 0 { writeTimeout = 10 * time.Second }

	return &Config{
		Env:                viper.GetString("env"),
		HTTPAddr:           viper.GetString("http.addr"),
		ReadTimeout:        readTimeout,
		WriteTimeout:       writeTimeout,
		KafkaBrokers:       viper.GetStringSlice("kafka.brokers"),
		KafkaInputTopic:    viper.GetString("kafka.input_topic"),
		KafkaOutputTopic:   viper.GetString("kafka.output_topic"),
		KafkaDLQTopic:      viper.GetString("kafka.dlq_topic"),
		KafkaConsumerGroup: viper.GetString("kafka.consumer_group"),
		KafkaWorkerCount:   viper.GetInt("kafka.worker_count"),
		RedisURL:           viper.GetString("redis.url"),
		OTelEndpoint:       viper.GetString("otel.endpoint"),
		SchemaVersion:      viper.GetString("schema.version"),
		PagerDutyAPIKey:    viper.GetString("pagerduty.api_key"),
		PagerDutyBaseURL:   viper.GetString("pagerduty.base_url"),
		TwilioAccountSID:   viper.GetString("twilio.account_sid"),
		TwilioAuthToken:    viper.GetString("twilio.auth_token"),
		TwilioFromNumber:   viper.GetString("twilio.from_number"),
		TwilioBaseURL:      viper.GetString("twilio.base_url"),
		APNsKeyPath:        viper.GetString("apns.key_path"),
		APNsTeamID:         viper.GetString("apns.team_id"),
		APNsBundleID:       viper.GetString("apns.bundle_id"),
		FCMServerKey:       viper.GetString("fcm.server_key"),
		CBFailureThreshold: viper.GetInt("cb.failure_threshold"),
		CBRecoverySeconds:  viper.GetInt("cb.recovery_seconds"),
		MaxRetries:         viper.GetInt("max_retries"),
		DegradedMode:       viper.GetBool("degraded_mode"),
	}
}
