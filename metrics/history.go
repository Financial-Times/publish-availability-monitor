package metrics

import "sync"

type History struct {
	sync.RWMutex
	PublishMetrics []PublishMetric
}

func NewHistory(metrics []PublishMetric) *History {
	return &History{
		sync.RWMutex{},
		metrics,
	}
}
