package config

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	log "github.com/Sirupsen/logrus"
)

// AppConfig holds the application's configuration
type AppConfig struct {
	Threshold           int                  `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf           consumer.QueueConfig `json:"queueConfig"`
	MetricConf          []MetricConfig       `json:"metricConfig"`
	SplunkConf          SplunkConfig         `json:"splunk-config"`
	HealthConf          HealthConfig         `json:"healthConfig"`
	ValidationEndpoints map[string]string    `json:"validationEndpoints"` //contentType to validation endpoint mapping, ex. { "EOM::Story": "http://methode-article-transformer/content-transform" }
	UUIDResolverURL     string               `json:"uuidResolverUrl"`
	Capabilities        []Capability         `json:"capabilities"`
}

// MetricConfig is the configuration of a PublishMetric
type MetricConfig struct {
	Granularity  int      `json:"granularity"` //how we split up the threshold, ex. 120/12
	Endpoint     string   `json:"endpoint"`
	ContentTypes []string `json:"contentTypes"` //list of valid eom types for this metric
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
func NewAppConfig(configFileName string) (*AppConfig, error) {
	file, err := ioutil.ReadFile(configFileName)
	if err != nil {
		log.Errorf("Error reading configuration file [%v]: [%v]", configFileName, err.Error())
		return nil, err
	}

	var conf AppConfig
	err = json.Unmarshal(file, &conf)
	if err != nil {
		log.Errorf("Error unmarshalling configuration file [%v]: [%v]", configFileName, err.Error())
		return nil, err
	}

	return &conf, nil
}

func (cfg *AppConfig) IsCapabilityMetric(alias string) bool {
	for _, c := range cfg.Capabilities {
		if c.MetricAlias == alias {
			return true
		}
	}

	return false
}

func IsE2ETestTransactionID(tid string, e2eTestUUIDs []string) bool {
	for _, testUUID := range e2eTestUUIDs {
		if strings.Contains(tid, testUUID) {
			return true
		}
	}

	return false
}
