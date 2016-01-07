package fthealth

import (
	"time"
)

type CheckResult struct {
	BusinessImpact   string    `json:"businessImpact"`
	Output           string    `json:"checkOutput"`
	LastUpdated      time.Time `json:"lastUpdated"`
	Name             string    `json:"name"`
	Ok               bool      `json:"ok"`
	PanicGuide       string    `json:"panicGuide"`
	Severity         uint8     `json:"severity"`
	TechnicalSummary string    `json:"technicalSummary"`
}

type HealthResult struct {
	Checks        []CheckResult `json:"checks"`
	Description   string        `json:"description"`
	Name          string        `json:"name"`
	SchemaVersion float64       `json:"schemaVersion"`
	Ok            bool          `json:"ok"`
	Severity      uint8         `json:"severity,omitempty"`
}
