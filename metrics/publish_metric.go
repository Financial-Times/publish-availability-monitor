package metrics

import (
	"fmt"
	"net/url"
	"time"
)

// PublishMetric holds the information about the metric we are measuring.
type PublishMetric struct {
	UUID            string
	PublishOK       bool      //did it meet the SLA?
	PublishDate     time.Time //the time WE get the message
	Platform        string
	PublishInterval Interval //the interval it was actually published in, ex. (10,20)
	Config          Config
	Endpoint        url.URL
	TID             string
	IsMarkedDeleted bool
}

func (pm PublishMetric) String() string {
	return fmt.Sprintf("Tid: %s, UUID: %s, Platform: %s, Endpoint: %s, PublishDate: %s, Duration: %d, Succeeded: %t.",
		pm.TID,
		pm.UUID,
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

// Config is the configuration of a PublishMetric
type Config struct {
	Granularity  int      `json:"granularity"` //how we split up the threshold, ex. 120/12
	Endpoint     string   `json:"endpoint"`
	ContentTypes []string `json:"contentTypes"` //list of valid eom types for this metric
	Alias        string   `json:"alias"`
	Health       string   `json:"health,omitempty"`
	APIKey       string   `json:"apiKey,omitempty"`
}
