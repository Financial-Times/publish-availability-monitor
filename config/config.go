package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	log "github.com/Sirupsen/logrus"
)

// AppConfig holds the application's configuration
type AppConfig struct {
	Threshold           int                  `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf           consumer.QueueConfig `json:"queueConfig"`
	MetricConf          []metrics.Config     `json:"metricConfig"`
	SplunkConf          SplunkConfig         `json:"splunk-config"`
	HealthConf          HealthConfig         `json:"healthConfig"`
	ValidationEndpoints map[string]string    `json:"validationEndpoints"` //contentType to validation endpoint mapping, ex. { "EOM::Story": "http://methode-article-transformer/content-transform" }
	UUIDResolverURL     string               `json:"uuidResolverUrl"`
	Capabilities        []Capability         `json:"capabilities"`
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
