package metrics

import "sync"

type PublishMetricsHistory struct {
	sync.RWMutex
	PublishMetrics []PublishMetric
}

func NewPublishMetricsHistory(metrics []PublishMetric) *PublishMetricsHistory {
	return &PublishMetricsHistory{
		sync.RWMutex{},
		metrics,
	}
}
