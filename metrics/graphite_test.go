package metrics

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/config"
)

func TestGraphiteSend(t *testing.T) {
	tests := map[string]struct {
		AppConfig            *config.AppConfig
		PublishOk            bool
		ExpectedStatusPrefix string
		ExpectedTimePrefix   string
		ExpectTimeout        bool
	}{
		"capability metric success should be sent to graphite": {
			AppConfig: &config.AppConfig{
				MetricConf: []config.MetricConfig{
					{
						Alias: "test-metric",
					},
				},
				Capabilities: []config.Capability{
					{
						Name:        "test-capability",
						MetricAlias: "test-metric",
					},
				},
				GraphiteUUID: "e435d5ef-20da-4c61-928c-71bfbec9fa3e",
				Environment:  "test",
			},
			PublishOk:            true,
			ExpectedStatusPrefix: "e435d5ef-20da-4c61-928c-71bfbec9fa3e.test-capability.test.status 1",
			ExpectedTimePrefix:   "e435d5ef-20da-4c61-928c-71bfbec9fa3e.test-capability.test.time 3",
		},
		"capability metric fail should be sent to graphite": {
			AppConfig: &config.AppConfig{
				MetricConf: []config.MetricConfig{
					{
						Alias: "test-metric",
					},
				},
				Capabilities: []config.Capability{
					{
						Name:        "test-capability",
						MetricAlias: "test-metric",
					},
				},
				GraphiteUUID: "e435d5ef-20da-4c61-928c-71bfbec9fa3e",
				Environment:  "test",
			},
			PublishOk:            false,
			ExpectedStatusPrefix: "e435d5ef-20da-4c61-928c-71bfbec9fa3e.test-capability.test.status 0",
			ExpectedTimePrefix:   "e435d5ef-20da-4c61-928c-71bfbec9fa3e.test-capability.test.time 3",
		},
		"regular metric should not be sent to graphite": {
			AppConfig: &config.AppConfig{
				MetricConf: []config.MetricConfig{
					{
						Alias: "test-metric",
					},
				},
				GraphiteUUID: "e435d5ef-20da-4c61-928c-71bfbec9fa3e",
				Environment:  "test",
			},
			ExpectTimeout: true,
		},
	}

	log := logger.NewUPPLogger("test", "PANIC")

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer srv.Close()

			test.AppConfig.GraphiteAddress = srv.Addr().String()

			sender := NewGraphiteSender(test.AppConfig, log)
			metricConfig := test.AppConfig.MetricConf[0]
			metric := PublishMetric{
				PublishOK:       test.PublishOk,
				Config:          metricConfig,
				Capability:      test.AppConfig.GetCapability(metricConfig.Alias),
				PublishInterval: Interval{UpperBound: 3},
			}

			go sender.Send(metric)

			var resultCh = make(chan []byte)
			var errCh = make(chan error)
			go func() {
				conn, err := srv.Accept()
				if err != nil {
					errCh <- err
					return
				}
				defer conn.Close()

				buf, err := io.ReadAll(conn)
				if err != nil {
					errCh <- err
					return
				}
				resultCh <- buf
			}()

			select {
			case result := <-resultCh:
				metrics := strings.FieldsFunc(string(result), func(c rune) bool {
					return c == '\n'
				})

				statusMetric, timeMetric := metrics[0], metrics[1]
				if !strings.HasPrefix(statusMetric, test.ExpectedStatusPrefix) {
					t.Fatalf("expected metric with prefix %v, got %v", test.ExpectedStatusPrefix, statusMetric)
				}

				if !strings.HasPrefix(timeMetric, test.ExpectedTimePrefix) {
					t.Fatalf("expected metric with prefix %v, got %v", test.ExpectedTimePrefix, timeMetric)
				}
			case err := <-errCh:
				t.Fatal(err)
			case <-time.After(1 * time.Second):
				if !test.ExpectTimeout {
					t.Fatalf("test timed out while sending metric")
				}
			}
		})
	}
}
