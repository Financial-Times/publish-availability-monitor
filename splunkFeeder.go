package main

import (
	"log"
	"os"
)

// SplunkFeeder implements MetricDestination interface to send PublishMetrics to Splunk.
// This is achieved by writing the metric into a file which is indexed by Splunk.
type SplunkFeeder struct {
	MetricLog *log.Logger
}

// NewSplunkFeeder returns a SplunkFeeder which will write the PublishMetrics to the file at filePath.
// If the file exists, it will be appended to.
func NewSplunkFeeder(logPrefix string) *SplunkFeeder {
	logger := log.New(os.Stdout, logPrefix, log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
	return &SplunkFeeder{logger}

}

// Send logs pm into a file.
func (sf SplunkFeeder) Send(pm PublishMetric) {
	sf.MetricLog.Printf("UUID=%v readEnv=%v transaction_id=%v publishDate=%v publishOk=%v duration=%v endpoint=%v ",
		pm.UUID, pm.platform, pm.tid, pm.publishDate.UnixNano(), pm.publishOK, pm.publishInterval.upperBound, pm.config.Alias)
}

type SplunkSLAFeeder interface {
	SendPublishSLA(sla SLAMetric)
}

type DefaultSplunkSLAFeeder struct {
	MetricLog *log.Logger
}

func NewSplunkSLAFeeder(logPrefix string) SplunkSLAFeeder {
	logger := log.New(os.Stdout, logPrefix, log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
	return DefaultSplunkSLAFeeder{logger}
}

func (sf DefaultSplunkSLAFeeder) SendPublishSLA(sm SLAMetric) {
	sf.MetricLog.Printf("UUID=%v metPublishSLA=%v okEnvironments=%v transaction_id=%v publishDate=%v", sm.UUID, sm.metPublishSLA, sm.okEnvironments, sm.tid, sm.publishDate.UnixNano())
}
