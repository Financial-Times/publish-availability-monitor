package main

import (
	"time"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"sort"
)

type ContentEvent struct {
	contentToCheck content.Content
	publishDate time.Time
	tid string
	isMarkedDeleted bool
}

type Collator interface {
	MeasureSLA(event ContentEvent)
}

type DefaultCollator struct {
	environments map[string]Environment
	scheduler Scheduler
	gtg GTGRunner
	aggregator chan PublishMetric
	slaFeeder SplunkSLAFeeder
}

type SLAMetric struct {
	UUID string
	tid string
	publishDate time.Time
	metPublishSLA bool
	okEnvironments []string
}

func (c DefaultCollator) MeasureSLA(event ContentEvent){
	context := c.scheduler.ScheduleChecks(event, c.environments)
	go c.CollateResults(event, context)
}

func (c DefaultCollator) CollateResults(event ContentEvent, context CheckContext){
	context.waitGroup.Wait()

	mappedResults := make(map[string] bool)
	for name := range c.environments {
		mappedResults[name] = true
	}

	close(context.results) // stop the channel, we don't expect anything else

	for check := range context.results {
		c.aggregator <- check // output splunk-style logging for the individual checks
		if !check.publishOK {
			mappedResults[check.platform] = false
		}
	}

	ignoreEnvironment := make(map[string] bool)
	okEnvironments := []string{}

	for name, healthy := range mappedResults {
		if !healthy { // At least one check failed for this environment and for this piece of content
			ignoreEnvironment[name] = c.gtg.RunGtgChecks(c.environments, c.environments[name])
		} else {
			okEnvironments = append(okEnvironments, name)
		}
	}

	sort.Strings(okEnvironments)

	slaMetric := SLAMetric{UUID: event.contentToCheck.GetUUID(), tid: event.tid, publishDate: event.publishDate, okEnvironments: okEnvironments, metPublishSLA: true}

	if len(ignoreEnvironment) == len(c.environments) { // Number of unhealthy environments == the total number of environments
		slaMetric.metPublishSLA = false
		c.slaFeeder.SendPublishSLA(slaMetric)
		return
	}

	for _, ignore := range ignoreEnvironment {
		if !ignore { // If anything should not be ignored, then SLA false.
			slaMetric.metPublishSLA = false
			c.slaFeeder.SendPublishSLA(slaMetric)
			return
		}
	}

	c.slaFeeder.SendPublishSLA(slaMetric)
}