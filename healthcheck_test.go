package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	"github.com/stretchr/testify/assert"
)

func TestBuildFtHealthcheckUrl(t *testing.T) {
	var testCases = []struct {
		validationURL     string
		health            string
		expectedHealthURL string
	}{
		{
			validationURL:     "http://example-service/validate/",
			health:            "/__example-service/__health",
			expectedHealthURL: "http://example-service/__example-service/__health",
		},
		{
			validationURL:     "http://example-service/validate?monitor=true",
			health:            "/__example-service/__health",
			expectedHealthURL: "http://example-service/__example-service/__health",
		},
	}
	for _, tc := range testCases {
		uri, _ := url.Parse(tc.validationURL)
		if actual, _ := buildFtHealthcheckUrl(*uri, tc.health); actual != tc.expectedHealthURL {
			t.Errorf("For [%s]:\n\tExpected: [%s]\n\tActual: [%s]", tc.validationURL, tc.expectedHealthURL, actual)
		}
	}
}

func TestPublishNoFailuresForSameUUIDs(t *testing.T) {
	metricConfig := config.MetricConfig{}
	interval := metrics.Interval{LowerBound: 5, UpperBound: 5}
	newUrl := url.URL{}
	t0 := time.Now()
	publishMetric1 := metrics.PublishMetric{
		UUID:            "1234567",
		PublishOK:       false,
		PublishDate:     t0,
		Platform:        "",
		PublishInterval: interval,
		Config:          metricConfig,
		Endpoint:        newUrl,
		TID:             "tid_1234",
		IsMarkedDeleted: false,
	}

	publishMetric2 := metrics.PublishMetric{
		UUID:            "1234567",
		PublishOK:       false,
		PublishDate:     t0,
		Platform:        "",
		PublishInterval: interval,
		Config:          metricConfig,
		Endpoint:        newUrl,
		TID:             "tid_6789",
		IsMarkedDeleted: false,
	}

	publishMetric3 := metrics.PublishMetric{
		UUID:            "1234567",
		PublishOK:       false,
		PublishDate:     t0,
		Platform:        "",
		PublishInterval: interval,
		Config:          metricConfig,
		Endpoint:        newUrl,
		TID:             "tid_6789",
		IsMarkedDeleted: false,
	}

	testMetrics := []metrics.PublishMetric{publishMetric1, publishMetric2, publishMetric3}
	testPublishHistory := metrics.NewHistory(testMetrics)

	testHealthcheck := Healthcheck{
		config:          &config.AppConfig{},
		metricContainer: testPublishHistory,
	}
	_, err := testHealthcheck.checkForPublishFailures()

	assert.NoError(t, err, "No Error expected if multiple fails for the same uuid")
}

func TestPublishFailureForDistinctUUIDs(t *testing.T) {
	metricConfig := config.MetricConfig{}
	interval := metrics.Interval{LowerBound: 5, UpperBound: 5}
	newUrl := url.URL{}
	t0 := time.Now()
	publishMetric1 := metrics.PublishMetric{
		UUID:            "12345",
		PublishOK:       false,
		PublishDate:     t0,
		Platform:        "",
		PublishInterval: interval,
		Config:          metricConfig,
		Endpoint:        newUrl,
		TID:             "tid_1234",
		IsMarkedDeleted: false,
	}

	publishMetric2 := metrics.PublishMetric{
		UUID:            "12678",
		PublishOK:       false,
		PublishDate:     t0,
		Platform:        "",
		PublishInterval: interval,
		Config:          metricConfig,
		Endpoint:        newUrl,
		TID:             "tid_6789",
		IsMarkedDeleted: false,
	}

	publishMetric3 := metrics.PublishMetric{
		UUID:            "12679",
		PublishOK:       false,
		PublishDate:     t0,
		Platform:        "",
		PublishInterval: interval,
		Config:          metricConfig,
		Endpoint:        newUrl,
		TID:             "tid_6789",
		IsMarkedDeleted: false,
	}

	testMetrics := []metrics.PublishMetric{publishMetric1, publishMetric2, publishMetric3}
	testPublishHistory := metrics.NewHistory(testMetrics)

	testHealthcheck := Healthcheck{
		config:          &config.AppConfig{},
		metricContainer: testPublishHistory,
	}
	_, err := testHealthcheck.checkForPublishFailures()

	assert.Error(t, err, "Expected Error for at least two distinct uuid publish fails")
}
