package metrics

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
}

// NewAggregator returns an Aggregator which reads metrics from inputChannel and
// distributes them to destinations.
func NewAggregator(inputChannel chan PublishMetric, destinations []Destination) *Aggregator {
	return &Aggregator{inputChannel, destinations}
}

// Run reads PublishMetrics from a channel and distributes them to a list of
// Destinations.
// Stops reading when the channel is closed.
func (a *Aggregator) Run() {
	for publishMetric := range a.publishMetricSource {
		for _, sender := range a.publishMetricDestinations {
			go sender.Send(publishMetric)
		}
	}
}
