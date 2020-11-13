package metrics

import (
	"github.com/Financial-Times/publish-availability-monitor/config"
	log "github.com/Sirupsen/logrus"
)

// Destination is the interface which defines a method to send
// PublishMetrics to a certain destination.
type Destination interface {
	Send(pm PublishMetric)
}

// Aggregator reads PublishMetrics from a channel and distributes them to
// the configured Destinations.
type Aggregator struct {
	publishMetricSource       chan PublishMetric
	publishMetricDestinations []Destination
	e2eTestUUIDs              []string
}

// NewAggregator returns an Aggregator which reads metrics from inputChannel and
// distributes them to destinations.
func NewAggregator(inputChannel chan PublishMetric, destinations []Destination, e2eTestUUIDs []string) *Aggregator {
	return &Aggregator{inputChannel, destinations, e2eTestUUIDs}
}

// Run reads PublishMetrics from a channel and distributes them to a list of
// Destinations.
// Stops reading when the channel is closed.
func (a *Aggregator) Run() {
	for publishMetric := range a.publishMetricSource {
		if config.IsE2ETestTransactionID(publishMetric.TID, a.e2eTestUUIDs) {
			log.Warnf("Got a E2E metric [%s] in aggregator, skipping ...", publishMetric.String())
			continue
		}

		for _, sender := range a.publishMetricDestinations {
			go sender.Send(publishMetric)
		}
	}
}
