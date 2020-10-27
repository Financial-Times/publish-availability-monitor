package config

import (
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/models"
)

// AppConfig holds the application's configuration
type AppConfig struct {
	Threshold           int                   `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf           consumer.QueueConfig  `json:"queueConfig"`
	MetricConf          []models.MetricConfig `json:"metricConfig"`
	SplunkConf          SplunkConfig          `json:"splunk-config"`
	HealthConf          HealthConfig          `json:"healthConfig"`
	ValidationEndpoints map[string]string     `json:"validationEndpoints"` //contentType to validation endpoint mapping, ex. { "EOM::Story": "http://methode-article-transformer/content-transform" }
	UUIDResolverURL     string                `json:"uuidResolverUrl"`
}

// SplunkConfig holds the SplunkFeeder-specific configuration
type SplunkConfig struct {
	LogPrefix string `json:"logPrefix"`
}

// HealthConfig holds the application's healthchecks configuration
type HealthConfig struct {
	FailureThreshold int `json:"failureThreshold"`
}
