package metrics

import (
	"sync"
	"testing"

	"github.com/Financial-Times/publish-availability-monitor/config"
)

func TestAggregatorRun(t *testing.T) {
	tests := map[string]struct {
		Metric                         PublishMetric
		ExpectedPublishMetricsCount    int
		ExpectedCapabilityMetricsCount int
	}{
		"regular metric should be sent only to publish metric destinations": {
			Metric: PublishMetric{
				Capability: nil,
			},
			ExpectedPublishMetricsCount:    1,
			ExpectedCapabilityMetricsCount: 0,
		},
		"capability metric should be sent only to capability destinations": {
			Metric: PublishMetric{
				Capability: &config.Capability{
					Name: "test-capability",
				},
			},
			ExpectedPublishMetricsCount:    0,
			ExpectedCapabilityMetricsCount: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(test.ExpectedPublishMetricsCount + test.ExpectedCapabilityMetricsCount)

			var metricsCh = make(chan PublishMetric)
			publishMetricDestination := &mockDestination{waitGroup: &wg}
			capabilityMetricDestination := &mockDestination{waitGroup: &wg}

			aggregator := NewAggregator(metricsCh,
				[]Destination{publishMetricDestination},
				[]Destination{capabilityMetricDestination})

			go aggregator.Run()
			metricsCh <- test.Metric
			wg.Wait()

			if len(publishMetricDestination.metrics) != test.ExpectedPublishMetricsCount {
				t.Fatalf("expected %v publish metrics, got %v", test.ExpectedPublishMetricsCount, len(publishMetricDestination.metrics))
			}

			if len(capabilityMetricDestination.metrics) != test.ExpectedCapabilityMetricsCount {
				t.Fatalf("expected %v capability metrics, got %v", test.ExpectedCapabilityMetricsCount, len(capabilityMetricDestination.metrics))
			}
		})
	}
}

type mockDestination struct {
	metrics   []PublishMetric
	waitGroup *sync.WaitGroup
}

func (md *mockDestination) Send(pm PublishMetric) {
	defer md.waitGroup.Done()
	md.metrics = append(md.metrics, pm)
}
