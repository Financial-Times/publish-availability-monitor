package metrics

import (
	"github.com/Financial-Times/go-logger/v2"
)

// Destination is the interface which defines a method to send
// PublishMetrics to a certain destination.
type Destination interface {
	Send(pm PublishMetric)
}

// Aggregator reads PublishMetrics from a channel and distributes them to
// the configured Destinations.
type Aggregator struct {
	publishMetricSource          chan PublishMetric
	publishMetricDestinations    []Destination
	capabilityMetricDestinations []Destination
	log                          *logger.UPPLogger
}

// NewAggregator returns an Aggregator which reads metrics from inputChannel and
// distributes them to destinations.
func NewAggregator(inputChannel chan PublishMetric, publishMetricDestinations, capabilityMetricDestinations []Destination, log *logger.UPPLogger) *Aggregator {
	return &Aggregator{
		publishMetricSource:          inputChannel,
		publishMetricDestinations:    publishMetricDestinations,
		capabilityMetricDestinations: capabilityMetricDestinations,
		log:                          log,
	}
}

// Run reads PublishMetrics from a channel and distributes them to a list of
// Destinations.
// Stops reading when the channel is closed.
func (a *Aggregator) Run() {
	for publishMetric := range a.publishMetricSource {
		if publishMetric.Capability != nil {
			a.log.Infof("Got a E2E metric [%s] in aggregator", publishMetric.String())
			for _, sender := range a.capabilityMetricDestinations {
				go sender.Send(publishMetric)
			}

			continue
		}

		for _, sender := range a.publishMetricDestinations {
			go sender.Send(publishMetric)
		}
	}
}
