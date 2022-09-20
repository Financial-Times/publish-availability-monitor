package config

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/Financial-Times/go-logger/v2"
)

// AppConfig holds the application's configuration
type AppConfig struct {
	Threshold           int               `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf           QueueConfig       `json:"queueConfig"`
	MetricConf          []MetricConfig    `json:"metricConfig"`
	SplunkConf          SplunkConfig      `json:"splunk-config"`
	HealthConf          HealthConfig      `json:"healthConfig"`
	ValidationEndpoints map[string]string `json:"validationEndpoints"` //contentType to validation endpoint mapping
	Capabilities        []Capability      `json:"capabilities"`
	GraphiteAddress     string            `json:"graphiteAddress"`
	GraphiteUUID        string            `json:"graphiteUUID"`
	Environment         string            `json:"environment"`
}

// QueueConfig is the configuration for kafka consumer queue
type QueueConfig struct {
	ConsumerGroup           string `json:"consumerGroup"`
	KafkaLagTolerance       int    `json:"lagTolerance"`
	Topic                   string `json:"topic"`
	BrokersConnectionString string `json:"connectionString"`
}

// MetricConfig is the configuration of a PublishMetric
type MetricConfig struct {
	Granularity  int      `json:"granularity"` //how we split up the threshold, ex. 120/12
	Endpoint     string   `json:"endpoint"`
	ContentTypes []string `json:"contentTypes"` //list of valid types for this metric
	Alias        string   `json:"alias"`
	Health       string   `json:"health,omitempty"`
	APIKey       string   `json:"apiKey,omitempty"`
}

// SplunkConfig holds the SplunkFeeder-specific configuration
type SplunkConfig struct {
	LogPrefix string `json:"logPrefix"`
}

// HealthConfig holds the application's healthchecks configuration
type HealthConfig struct {
	FailureThreshold int `json:"failureThreshold"`
}

// Capability represents business capability configuration
type Capability struct {
	Name        string   `json:"name"`
	MetricAlias string   `json:"metricAlias"`
	TestIDs     []string `json:"testIDs"`
}

// NewAppConfig opens the file at configFileName and unmarshals it into an AppConfig.
func NewAppConfig(configFileName string, log *logger.UPPLogger) (*AppConfig, error) {
	file, err := ioutil.ReadFile(configFileName)
	if err != nil {
		log.WithError(err).Errorf("Error reading configuration file [%v]", configFileName)
		return nil, err
	}

	var conf AppConfig
	err = json.Unmarshal(file, &conf)
	if err != nil {
		log.WithError(err).Errorf("Error unmarshalling configuration file [%v]", configFileName)
		return nil, err
	}

	return &conf, nil
}

func (cfg *AppConfig) GetCapability(metricAlias string) *Capability {
	for _, c := range cfg.Capabilities {
		if c.MetricAlias == metricAlias {
			return &c
		}
	}

	return nil
}

func IsE2ETestTransactionID(tid string, e2eTestUUIDs []string) bool {
	for _, testUUID := range e2eTestUUIDs {
		if strings.Contains(tid, testUUID) {
			return true
		}
	}

	return false
}
