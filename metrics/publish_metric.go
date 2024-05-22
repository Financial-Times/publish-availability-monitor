package metrics

import (
	"fmt"
	"net/url"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/config"
)

// PublishMetric holds the information about the metric we are measuring.
type PublishMetric struct {
	UUID            string
	EditorialDesk   string
	Publication     []string
	PublishOK       bool      // did it meet the SLA?
	PublishDate     time.Time // the time WE get the message
	Platform        string
	PublishInterval Interval // the interval it was actually published in, ex. (10,20)
	Config          config.MetricConfig
	Endpoint        url.URL
	TID             string
	IsMarkedDeleted bool
	Capability      *config.Capability
}

func (pm PublishMetric) String() string {
	return fmt.Sprintf(
		"Tid: %s, UUID: %s, Editorial Desk: %s, Publication %v, Platform: %s, Endpoint: %s, PublishDate: %s, Duration: %d, Succeeded: %t.",
		pm.TID,
		pm.UUID,
		pm.EditorialDesk,
		pm.Publication,
		pm.Platform,
		pm.Config.Alias,
		pm.PublishDate.String(),
		pm.PublishInterval.UpperBound,
		pm.PublishOK,
	)
}

// Interval is a simple representation of an interval of time, with a lower and
// upper boundary
type Interval struct {
	LowerBound int
	UpperBound int
}
