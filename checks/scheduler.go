package checks

import (
	"net/url"
	"regexp"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	log "github.com/Sirupsen/logrus"
)

var (
	AbsoluteURLRegex = regexp.MustCompile("(?i)https?://.*")
)

type SchedulerParam struct {
	contentToCheck  content.Content
	publishDate     time.Time
	tid             string
	isMarkedDeleted bool
	metricContainer *metrics.History
	environments    *envs.Environments
}

func ScheduleChecks(p *SchedulerParam, subscribedFeeds map[string][]feeds.Feed, endpointSpecificChecks map[string]EndpointSpecificCheck, appConfig *config.AppConfig, metricSink chan metrics.PublishMetric, e2eTestUUIDs []string) {
	isE2ETest := config.IsE2ETestTransactionID(p.tid, e2eTestUUIDs)

	for _, metric := range appConfig.MetricConf {
		if !validType(metric.ContentTypes, p.contentToCheck.GetType()) && !isE2ETest {
			continue
		}

		var capability *config.Capability
		if isE2ETest {
			capability = appConfig.GetCapability(metric.Alias)
			if capability == nil {
				continue
			}
		}

		if p.environments.Len() > 0 {
			for _, name := range p.environments.Names() {
				env := p.environments.Environment(name)
				var endpointURL *url.URL
				var err error

				if AbsoluteURLRegex.MatchString(metric.Endpoint) {
					endpointURL, err = url.Parse(metric.Endpoint)
				} else {
					if metric.Alias == "S3" {
						endpointURL, err = url.Parse(env.S3Url + metric.Endpoint)
					} else {
						endpointURL, err = url.Parse(env.ReadURL + metric.Endpoint)
					}
				}

				if err != nil {
					log.Errorf("Cannot parse url [%v], error: [%v]", metric.Endpoint, err.Error())
					continue
				}

				var publishMetric = metrics.PublishMetric{
					UUID:            p.contentToCheck.GetUUID(),
					PublishOK:       false,
					PublishDate:     p.publishDate,
					Platform:        name,
					PublishInterval: metrics.Interval{},
					Config:          metric,
					Endpoint:        *endpointURL,
					TID:             p.tid,
					IsMarkedDeleted: p.isMarkedDeleted,
					Capability:      capability,
				}

				var checkInterval = appConfig.Threshold / metric.Granularity
				var publishCheck = NewPublishCheck(publishMetric, env.Username, env.Password, appConfig.Threshold, checkInterval, metricSink, endpointSpecificChecks)
				go scheduleCheck(*publishCheck, p.metricContainer)
			}
		} else {
			// generate a generic failure metric so that the absence of monitoring is logged
			var publishMetric = metrics.PublishMetric{
				UUID:            p.contentToCheck.GetUUID(),
				PublishOK:       false,
				PublishDate:     p.publishDate,
				Platform:        "none",
				PublishInterval: metrics.Interval{},
				Config:          metric,
				Endpoint:        url.URL{},
				TID:             p.tid,
				IsMarkedDeleted: p.isMarkedDeleted,
				Capability:      capability,
			}
			metricSink <- publishMetric
			p.metricContainer.Update(publishMetric)
		}
	}
}

func scheduleCheck(check PublishCheck, metricContainer *metrics.History) {
	//the date the SLA expires for this publish event
	publishSLA := check.Metric.PublishDate.Add(time.Duration(check.Threshold) * time.Second)

	//compute the actual seconds left until the SLA to compensate for the
	//time passed between publish and the message reaching this point
	secondsUntilSLA := publishSLA.Sub(time.Now()).Seconds()
	log.Infof("Checking %s. [%v] seconds until SLA.",
		LoggingContextForCheck(check.Metric.Config.Alias,
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
		LoggingContextForCheck(check.Metric.Config.Alias,
			check.Metric.UUID,
			check.Metric.Platform,
			check.Metric.TID),
		int(secondsSincePublish))

	elapsedIntervals := secondsSincePublish / float64(check.CheckInterval)
	log.Infof("Checking %s. Skipping first [%v] checks",
		LoggingContextForCheck(check.Metric.Config.Alias,
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
				LoggingContextForCheck(check.Metric.Config.Alias,
					check.Metric.UUID,
					check.Metric.Platform,
					check.Metric.TID))
			tickerChan.Stop()
			return
		}

		lower := (checkNr - 1) * check.CheckInterval
		upper := checkNr * check.CheckInterval
		check.Metric.PublishInterval = metrics.Interval{
			LowerBound: lower,
			UpperBound: upper,
		}

		if checkSuccessful {
			tickerChan.Stop()
			check.Metric.PublishOK = true

			check.ResultSink <- check.Metric
			metricContainer.Update(check.Metric)
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
			metricContainer.Update(check.Metric)
			return
		}
	}

}

func validType(validTypes []string, typeToCheck string) bool {
	for _, t := range validTypes {
		if t == typeToCheck {
			return true
		}
	}
	return false
}
