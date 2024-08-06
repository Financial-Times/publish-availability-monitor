package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
)

func TestHandleMessage_ProducesMetrics(t *testing.T) {
	log := logger.NewUPPLogger("publish-availability-monitor", "INFO")
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	tests := map[string]struct {
		AppConfig            *config.AppConfig
		PublicationConfig    *config.PublicationsConfig
		E2ETestUUIDs         []string
		KafkaMessage         kafka.FTMessage
		NotificationsPayload string
		IsMetricExpected     bool
		ExpectedMetric       metrics.PublishMetric
	}{
		"synthetic e2e test with valid content type and notification should result in publishOk=true": {
			AppConfig: &config.AppConfig{
				Threshold: 5,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal+json": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:    "/whatever/",
						Granularity: 1,
						Alias:       "notifications-push",
					},
				},
				Capabilities: []config.Capability{
					{
						MetricAlias: "notifications-push",
					},
				},
				NotificationsPushPublicationMonitorList: "88fdde6c-2aa4-4f78-af02-9f680097cfd6",
			},
			E2ETestUUIDs: []string{"077f5ac2-0491-420e-a5d0-982e0f86204b"},
			KafkaMessage: kafka.FTMessage{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal+json",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
				}`,
			},
			NotificationsPayload: getNotificationsPayload(
				"077f5ac2-0491-420e-a5d0-982e0f86204b",
				"SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
			),
			IsMetricExpected: true,
			ExpectedMetric: metrics.PublishMetric{
				TID:       "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
				PublishOK: true,
			},
		},
		"synthetic publish should not produce metric": {
			AppConfig: &config.AppConfig{
				Threshold: 1,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal+json": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:    "/whatever/",
						Granularity: 1,
						Alias:       "notifications-push",
					},
				},
				NotificationsPushPublicationMonitorList: "88fdde6c-2aa4-4f78-af02-9f680097cfd6",
			},
			KafkaMessage: kafka.FTMessage{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal+json",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article",
					"publication": ["88fdde6c-2aa4-4f78-af02-9f680097cfd6"]
				}`,
			},
			IsMetricExpected: false,
		},
		"content with missing validator should not produce metric": {
			AppConfig: &config.AppConfig{
				Threshold: 1,
				MetricConf: []config.MetricConfig{
					{
						Endpoint:    "/whatever/",
						Granularity: 1,
						Alias:       "notifications-push",
					},
				},
				NotificationsPushPublicationMonitorList: "88fdde6c-2aa4-4f78-af02-9f680097cfd6",
			},
			KafkaMessage: kafka.FTMessage{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal+json",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article",
					"publication": ["88fdde6c-2aa4-4f78-af02-9f680097cfd6"]
				}`,
			},
			IsMetricExpected: false,
		},
		"content with validator and without metric content type should not produce metric": {
			AppConfig: &config.AppConfig{
				Threshold: 1,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal+json": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:    "/whatever/",
						Granularity: 1,
						Alias:       "notifications-push",
					},
				},
				NotificationsPushPublicationMonitorList: "88fdde6c-2aa4-4f78-af02-9f680097cfd6",
			},
			KafkaMessage: kafka.FTMessage{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal+json",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article",
					"publication": ["88fdde6c-2aa4-4f78-af02-9f680097cfd6"]
				}`,
			},
			IsMetricExpected: false,
		},
		"content with validator without notification should result in publishOk=false": {
			AppConfig: &config.AppConfig{
				Threshold: 5,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal+json": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:     "/whatever/",
						Granularity:  1,
						Alias:        "notifications-push",
						ContentTypes: []string{"application/vnd.ft-upp-article-internal+json"},
					},
				},
				NotificationsPushPublicationMonitorList: "88fdde6c-2aa4-4f78-af02-9f680097cfd6",
			},
			KafkaMessage: kafka.FTMessage{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal+json",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article",
					"publication": ["88fdde6c-2aa4-4f78-af02-9f680097cfd6"]
				}`,
			},
			IsMetricExpected: true,
			ExpectedMetric: metrics.PublishMetric{
				TID:       "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
				PublishOK: false,
			},
		},
		"content with validator, metric content type and notification should result in publishOk=true": {
			AppConfig: &config.AppConfig{
				Threshold: 5,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal+json": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:     "/whatever/",
						Granularity:  1,
						Alias:        "notifications-push",
						ContentTypes: []string{"application/vnd.ft-upp-article-internal+json"},
					},
				},
				NotificationsPushPublicationMonitorList: "88fdde6c-2aa4-4f78-af02-9f680097cfd6",
			},
			KafkaMessage: kafka.FTMessage{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal+json",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article",
					"publication": ["88fdde6c-2aa4-4f78-af02-9f680097cfd6"]
				}`,
			},
			NotificationsPayload: getNotificationsPayload(
				"077f5ac2-0491-420e-a5d0-982e0f86204b",
				"tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
			),
			IsMetricExpected: true,
			ExpectedMetric: metrics.PublishMetric{
				TID:       "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
				PublishOK: true,
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testEnvs := envs.NewEnvironments()
			testEnvs.SetEnvironment("env1", envs.Environment{
				Name:    "env1",
				ReadURL: "http://env1.example.org",
			})

			httpCaller := &mockHTTPCaller{
				notifications: []string{test.NotificationsPayload},
			}

			baseURL, _ := url.Parse("http://www.example.org")
			f := feeds.NewNotificationsFeed("notifications-push", *baseURL, &config.PublicationsConfig{}, 10, 1, "", "", "", log)
			f.(*feeds.NotificationsPushFeed).SetHTTPCaller(httpCaller)
			f.Start()
			defer f.Stop()

			subscribedFeeds := map[string][]feeds.Feed{}
			subscribedFeeds["env1"] = append(subscribedFeeds["env1"], f)

			metricsCh := make(chan metrics.PublishMetric)
			metricsHistory := metrics.NewHistory(make([]metrics.PublishMetric, 0))

			mh := NewKafkaMessageHandler(
				test.AppConfig,
				test.PublicationConfig,
				testEnvs,
				subscribedFeeds,
				metricsCh,
				metricsHistory,
				test.E2ETestUUIDs,
				log,
			)
			kmh := mh.(*kafkaMessageHandler)

			kmh.HandleMessage(test.KafkaMessage)

			timeout := time.Duration(test.AppConfig.Threshold) * time.Second
			metric, err := waitForMetric(metricsCh, timeout)
			if test.IsMetricExpected && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if !test.IsMetricExpected && err == nil {
				t.Fatal("expected non nil error")
			}

			if metric.TID != test.ExpectedMetric.TID {
				t.Fatalf("expected TID %v, got %v", test.ExpectedMetric.TID, metric.TID)
			}

			if metric.PublishOK != test.ExpectedMetric.PublishOK {
				t.Fatalf(
					"expected PublishOK %v, got %v",
					test.ExpectedMetric.PublishOK,
					metric.PublishOK,
				)
			}
		})
	}
}

func getNotificationsPayload(uuid, tID string) string {
	notifications := fmt.Sprintf(`{
		"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
		"id": "http://www.ft.com/thing/%v",
		"apiUrl": "http://api.ft.com/content/%v",
		"publishReference": "%v",
		"lastModified": "%v"
	}`, uuid, uuid, tID, time.Now().Format(time.RFC3339))

	return strings.Replace(notifications, "\n", "", -1)
}

func waitForMetric(
	metricsCh chan metrics.PublishMetric,
	timeout time.Duration,
) (metrics.PublishMetric, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			continue
		case metric := <-metricsCh:
			return metric, nil
		case <-timer.C:
			return metrics.PublishMetric{}, errors.New("test timed out waiting for metric")
		}
	}
}

type mockHTTPCaller struct {
	notifications []string
}

func (mht *mockHTTPCaller) DoCall(config httpcaller.Config) (*http.Response, error) {
	stream := &mockPushNotificationsStream{mht.notifications, 0}
	return &http.Response{
		StatusCode: 200,
		Body:       stream,
	}, nil
}

type mockPushNotificationsStream struct {
	notifications []string
	index         int
}

func (resp *mockPushNotificationsStream) Read(p []byte) (n int, err error) {
	var data []byte
	if resp.index >= len(resp.notifications) {
		data = []byte("data: []\n")
	} else {
		data = []byte("data: [" + resp.notifications[resp.index] + "]\n")
		resp.index++
	}
	actual := len(data)
	for i := 0; i < actual; i++ {
		p[i] = data[i]
	}

	return actual, nil
}

func (resp *mockPushNotificationsStream) Close() error {
	return nil
}
