package metrics

import (
	"fmt"
	"sync"
)

type History struct {
	mu             sync.RWMutex
	PublishMetrics []PublishMetric
}

func NewHistory(metrics []PublishMetric) *History {
	return &History{
		mu:             sync.RWMutex{},
		PublishMetrics: metrics,
	}
}

func (h *History) Update(newPublishResult PublishMetric) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.PublishMetrics) == 10 {
		h.PublishMetrics = h.PublishMetrics[1:len(h.PublishMetrics)]
	}
	h.PublishMetrics = append(h.PublishMetrics, newPublishResult)
}

func (h *History) GetFailures() map[string]struct{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	failures := make(map[string]struct{})
	var emptyStruct struct{}
	for i := 0; i < len(h.PublishMetrics); i++ {
		if !h.PublishMetrics[i].PublishOK {
			failures[h.PublishMetrics[i].UUID] = emptyStruct
		}
	}

	return failures
}

func (h *History) String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var s string
	for i := len(h.PublishMetrics) - 1; i >= 0; i-- {
		s += fmt.Sprintf("%d. %v\n\n", len(h.PublishMetrics)-i, h.PublishMetrics[i])
	}

	return s
}

func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.PublishMetrics)
}

func (h *History) First() *PublishMetric {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return &h.PublishMetrics[0]
}
