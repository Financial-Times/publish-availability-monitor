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

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	log "github.com/Sirupsen/logrus"
)

func TestHandleMessage_ProducesMetrics(t *testing.T) {
	log.SetLevel(log.PanicLevel)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := map[string]struct {
		AppConfig            *config.AppConfig
		E2ETestUUIDs         []string
		KafkaMessage         consumer.Message
		NotificationsPayload string
		IsMetricExpected     bool
		ExpectedMetric       metrics.PublishMetric
	}{
		"synthetic e2e test with valid content type and notification should result in publishOk=true": {
			AppConfig: &config.AppConfig{
				Threshold: 5,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal": testServer.URL,
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
			},
			E2ETestUUIDs: []string{"077f5ac2-0491-420e-a5d0-982e0f86204b"},
			KafkaMessage: consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
				}`,
			},
			NotificationsPayload: getNotificationsPayload("077f5ac2-0491-420e-a5d0-982e0f86204b", "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b"),
			IsMetricExpected:     true,
			ExpectedMetric: metrics.PublishMetric{
				TID:       "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
				PublishOK: true,
			},
		},
		"synthetic publish should not produce metric": {
			AppConfig: &config.AppConfig{
				Threshold: 1,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:    "/whatever/",
						Granularity: 1,
						Alias:       "notifications-push",
					},
				},
			},
			KafkaMessage: consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "SYNTHETIC-REQ-MON077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
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
			},
			KafkaMessage: consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
				}`,
			},
			IsMetricExpected: false,
		},
		"content with validator and without metric content type should not produce metric": {
			AppConfig: &config.AppConfig{
				Threshold: 1,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:    "/whatever/",
						Granularity: 1,
						Alias:       "notifications-push",
					},
				},
			},
			KafkaMessage: consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
				}`,
			},
			IsMetricExpected: false,
		},
		"content with validator without notification should result in publishOk=false": {
			AppConfig: &config.AppConfig{
				Threshold: 5,
				ValidationEndpoints: map[string]string{
					"application/vnd.ft-upp-article-internal": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:     "/whatever/",
						Granularity:  1,
						Alias:        "notifications-push",
						ContentTypes: []string{"application/vnd.ft-upp-article-internal"},
					},
				},
			},
			KafkaMessage: consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
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
					"application/vnd.ft-upp-article-internal": testServer.URL,
				},
				MetricConf: []config.MetricConfig{
					{
						Endpoint:     "/whatever/",
						Granularity:  1,
						Alias:        "notifications-push",
						ContentTypes: []string{"application/vnd.ft-upp-article-internal"},
					},
				},
			},
			KafkaMessage: consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id":  "http://cmdb.ft.com/systems/cct",
					"X-Request-Id":      "tid_077f5ac2-0491-420e-a5d0-982e0f86204b",
					"Content-Type":      "application/vnd.ft-upp-article-internal",
					"Message-Timestamp": time.Now().Format(checks.DateLayout),
				},
				Body: `{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"type": "Article"
				}`,
			},
			NotificationsPayload: getNotificationsPayload("077f5ac2-0491-420e-a5d0-982e0f86204b", "tid_077f5ac2-0491-420e-a5d0-982e0f86204b"),
			IsMetricExpected:     true,
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
			var testEnvs = envs.NewEnvironments()
			testEnvs.SetEnvironment("env1", envs.Environment{
				Name:    "env1",
				ReadURL: "http://env1.example.org",
			})

			httpCaller := &mockHTTPCaller{
				notifications: []string{test.NotificationsPayload},
			}

			baseURL, _ := url.Parse("http://www.example.org")
			f := feeds.NewNotificationsFeed("notifications-push", *baseURL, 10, 1, "", "", "")
			f.(*feeds.NotificationsPushFeed).SetHTTPCaller(httpCaller)
			f.Start()
			defer f.Stop()

			subscribedFeeds := map[string][]feeds.Feed{}
			subscribedFeeds["env1"] = append(subscribedFeeds["env1"], f)

			typeRes := new(MockTypeResolver)
			var metricsCh = make(chan metrics.PublishMetric)
			var metricsHistory = metrics.NewHistory(make([]metrics.PublishMetric, 0))

			mh := NewKafkaMessageHandler(typeRes, test.AppConfig, testEnvs, subscribedFeeds, metricsCh, metricsHistory, test.E2ETestUUIDs)
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
				t.Fatalf("expected PublishOK %v, got %v", test.ExpectedMetric.PublishOK, metric.PublishOK)
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

func waitForMetric(metricsCh chan metrics.PublishMetric, timeout time.Duration) (metrics.PublishMetric, error) {
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
