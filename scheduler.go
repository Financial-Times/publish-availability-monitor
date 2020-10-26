package main

import (
	"net/url"
	"regexp"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/models"
	log "github.com/Sirupsen/logrus"
)

var (
	absoluteUrlRegex = regexp.MustCompile("(?i)https?://.*")
)

type schedulerParam struct {
	contentToCheck  content.Content
	publishDate     time.Time
	tid             string
	isMarkedDeleted bool
	metricContainer *publishHistory
	environments    *threadSafeEnvironments
}

func scheduleChecks(p *schedulerParam, subscribedFeeds map[string][]feeds.Feed, endpointSpecificChecks map[string]checks.EndpointSpecificCheck) {
	for _, metric := range appConfig.MetricConf {
		if !validType(metric.ContentTypes, p.contentToCheck.GetType()) {
			continue
		}

		if p.environments.len() > 0 {
			for _, name := range p.environments.names() {
				env := p.environments.environment(name)
				var endpointURL *url.URL
				var err error

				if absoluteUrlRegex.MatchString(metric.Endpoint) {
					endpointURL, err = url.Parse(metric.Endpoint)
				} else {
					if metric.Alias == "S3" {
						endpointURL, err = url.Parse(env.S3Url + metric.Endpoint)
					} else {
						endpointURL, err = url.Parse(env.ReadUrl + metric.Endpoint)
					}
				}

				if err != nil {
					log.Errorf("Cannot parse url [%v], error: [%v]", metric.Endpoint, err.Error())
					continue
				}

				var publishMetric = models.PublishMetric{
					UUID:            p.contentToCheck.GetUUID(),
					PublishOK:       false,
					PublishDate:     p.publishDate,
					Platform:        name,
					PublishInterval: models.Interval{},
					Config:          metric,
					Endpoint:        *endpointURL,
					TID:             p.tid,
					IsMarkedDeleted: p.isMarkedDeleted,
				}

				var checkInterval = appConfig.Threshold / metric.Granularity
				var publishCheck = checks.NewPublishCheck(publishMetric, env.Username, env.Password, appConfig.Threshold, checkInterval, metricSink, endpointSpecificChecks)
				go scheduleCheck(*publishCheck, p.metricContainer)
			}
		} else {
			// generate a generic failure metric so that the absence of monitoring is logged
			var publishMetric = models.PublishMetric{
				UUID:            p.contentToCheck.GetUUID(),
				PublishOK:       false,
				PublishDate:     p.publishDate,
				Platform:        "none",
				PublishInterval: models.Interval{},
				Config:          metric,
				Endpoint:        url.URL{},
				TID:             p.tid,
				IsMarkedDeleted: p.isMarkedDeleted,
			}
			metricSink <- publishMetric
			updateHistory(p.metricContainer, publishMetric)
		}
	}
}

func scheduleCheck(check checks.PublishCheck, metricContainer *publishHistory) {
	//the date the SLA expires for this publish event
	publishSLA := check.Metric.PublishDate.Add(time.Duration(check.Threshold) * time.Second)

	//compute the actual seconds left until the SLA to compensate for the
	//time passed between publish and the message reaching this point
	secondsUntilSLA := publishSLA.Sub(time.Now()).Seconds()
	log.Infof("Checking %s. [%v] seconds until SLA.",
		checks.LoggingContextForCheck(check.Metric.Config.Alias,
			check.Metric.UUID,
			check.Metric.Platform,
			check.Metric.TID),
		int(secondsUntilSLA))

	//used to signal the ticker to stop after the threshold duration is reached
	quitChan := make(chan bool)
	go func() {
		<-time.After(time.Duration(secondsUntilSLA) * time.Second)
		close(quitChan)
	}()

	secondsSincePublish := time.Since(check.Metric.PublishDate).Seconds()
	log.Infof("Checking %s. [%v] seconds elapsed since publish.",
		checks.LoggingContextForCheck(check.Metric.Config.Alias,
			check.Metric.UUID,
			check.Metric.Platform,
			check.Metric.TID),
		int(secondsSincePublish))

	elapsedIntervals := secondsSincePublish / float64(check.CheckInterval)
	log.Infof("Checking %s. Skipping first [%v] checks",
		checks.LoggingContextForCheck(check.Metric.Config.Alias,
			check.Metric.UUID,
			check.Metric.Platform,
			check.Metric.TID),
		int(elapsedIntervals))

	checkNr := int(elapsedIntervals) + 1
	// ticker to fire once per interval
	tickerChan := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	for {
		checkSuccessful, ignoreCheck := check.DoCheck()
		if ignoreCheck {
			log.Infof("Ignore check for %s",
				checks.LoggingContextForCheck(check.Metric.Config.Alias,
					check.Metric.UUID,
					check.Metric.Platform,
					check.Metric.TID))
			tickerChan.Stop()
			return
		}
		if checkSuccessful {
			tickerChan.Stop()
			check.Metric.PublishOK = true

			lower := (checkNr - 1) * check.CheckInterval
			upper := checkNr * check.CheckInterval
			check.Metric.PublishInterval = models.Interval{
				LowerBound: lower,
				UpperBound: upper,
			}

			check.ResultSink <- check.Metric
			updateHistory(metricContainer, check.Metric)
			return
		}
		checkNr++
		select {
		case <-tickerChan.C:
			continue
		case <-quitChan:
			tickerChan.Stop()
			//if we get here, checks were unsuccessful
			check.Metric.PublishOK = false
			check.ResultSink <- check.Metric
			updateHistory(metricContainer, check.Metric)
			return
		}
	}

}

func updateHistory(metricContainer *publishHistory, newPublishResult models.PublishMetric) {
	metricContainer.Lock()
	if len(metricContainer.publishMetrics) == 10 {
		metricContainer.publishMetrics = metricContainer.publishMetrics[1:len(metricContainer.publishMetrics)]
	}
	metricContainer.publishMetrics = append(metricContainer.publishMetrics, newPublishResult)
	metricContainer.Unlock()
}

func validType(validTypes []string, eomType string) bool {
	for _, t := range validTypes {
		if t == eomType {
			return true
		}
	}
	return false
}
