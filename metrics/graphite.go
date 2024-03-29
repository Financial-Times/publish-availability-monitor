package metrics

import (
	"fmt"
	"net"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/config"
)

// GraphiteSender implements Destination interface to send PublishMetrics for capability E2E tests to Graphite.
type GraphiteSender struct {
	graphiteAddress string
	graphiteUUID    string
	environment     string
	log             *logger.UPPLogger
}

// NewGraphiteSender returns a GraphiteSender.
func NewGraphiteSender(cfg *config.AppConfig, log *logger.UPPLogger) *GraphiteSender {
	return &GraphiteSender{
		graphiteAddress: cfg.GraphiteAddress,
		graphiteUUID:    cfg.GraphiteUUID,
		environment:     cfg.Environment,
		log:             log,
	}
}

// Send transforms a Publish metric to Graphite one and sends it to Graphite endpoint.
func (gs *GraphiteSender) Send(pm PublishMetric) {
	if pm.Capability == nil {
		gs.log.Errorf("Cannot send non-capability metric %s to Graphite", pm.Config.Alias)
		return
	}

	metricPrefix := fmt.Sprintf("%s.%s.%s", gs.graphiteUUID, pm.Capability.Name, gs.environment)
	statusMetricName := fmt.Sprintf("%s.%s", metricPrefix, "status")
	var statusMetricValue int
	if pm.PublishOK {
		statusMetricValue = 1
	}
	statusMetric := fmt.Sprintf("%s %d %d\n", statusMetricName, statusMetricValue, time.Now().Unix())

	timeMetricName := fmt.Sprintf("%s.%s", metricPrefix, "time")
	timeMetricValue := pm.PublishInterval.UpperBound
	timeMetric := fmt.Sprintf("%s %d %d\n", timeMetricName, timeMetricValue, time.Now().Unix())

	conn, err := net.DialTimeout("tcp", gs.graphiteAddress, 10*time.Second)
	if err != nil {
		gs.log.WithError(err).Error("Cannot connect to Graphite")
		return
	}
	defer conn.Close()

	_, err = fmt.Fprint(conn, statusMetric, timeMetric)
	if err != nil {
		gs.log.WithError(err).Error("Cannot write Graphite metric")
	}
}
