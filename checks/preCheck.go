package checks

import (
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
)

type PreCheck func(
	publishedContent content.Content,
	tid string,
	publishDate time.Time,
	appConfig *config.AppConfig,
	metricContainer *metrics.History,
	environments *envs.Environments,
	log *logger.UPPLogger,
) (bool, *SchedulerParam)

func MainPreChecks() []PreCheck {
	return []PreCheck{mainPreCheck}
}

func mainPreCheck(
	publishedContent content.Content,
	tid string,
	publishDate time.Time,
	appConfig *config.AppConfig,
	metricContainer *metrics.History,
	environments *envs.Environments,
	log *logger.UPPLogger,
) (bool, *SchedulerParam) {
	uuid := publishedContent.GetUUID()
	validationEndpointKey := publishedContent.GetType()
	var validationEndpoint string
	var found bool
	var username string
	var password string

	if validationEndpoint, found = appConfig.ValidationEndpoints[validationEndpointKey]; found {
		username, password = envs.GetValidationCredentials()
	}

	logEntry := log.WithUUID(uuid).WithTransactionID(tid)

	valRes := publishedContent.Validate(validationEndpoint, tid, username, password, log)
	if !valRes.IsValid {
		logEntry.Info("Message is INVALID, skipping...")
		return false, nil
	}

	logEntry.Info("Message is VALID.")

	if isMessagePastPublishSLA(publishDate, appConfig.Threshold) {
		logEntry.Info("Message is past publish SLA, skipping.")
		return false, nil
	}

	return true, &SchedulerParam{
		contentToCheck:  publishedContent,
		publishDate:     publishDate,
		tid:             tid,
		isMarkedDeleted: valRes.IsMarkedDeleted,
		metricContainer: metricContainer,
		environments:    environments,
	}
}

func isMessagePastPublishSLA(date time.Time, threshold int) bool {
	passedSLA := date.Add(time.Duration(threshold) * time.Second)
	return time.Now().After(passedSLA)
}
