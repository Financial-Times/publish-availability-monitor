package checks

import (
	"time"

	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	log "github.com/Sirupsen/logrus"
)

func МainPreChecks() []func(publishedContent content.Content, tid string, publishDate time.Time, appConfig *config.AppConfig, metricContainer *metrics.History, environments *envs.Environments) (bool, *SchedulerParam) {
	return []func(publishedContent content.Content, tid string, publishDate time.Time, appConfig *config.AppConfig, metricContainer *metrics.History, environments *envs.Environments) (bool, *SchedulerParam){
		mainPreCheck,
	}
}

func mainPreCheck(publishedContent content.Content, tid string, publishDate time.Time, appConfig *config.AppConfig, metricContainer *metrics.History, environments *envs.Environments) (bool, *SchedulerParam) {
	uuid := publishedContent.GetUUID()
	validationEndpointKey := getValidationEndpointKey(publishedContent, tid, uuid)
	var validationEndpoint string
	var found bool
	var username string
	var password string

	if validationEndpoint, found = appConfig.ValidationEndpoints[validationEndpointKey]; found {
		username, password = envs.GetValidationCredentials()
	}

	valRes := publishedContent.Validate(validationEndpoint, tid, username, password)
	if !valRes.IsValid {
		log.Infof("Message [%v] with UUID [%v] is INVALID, skipping...", tid, uuid)
		return false, nil
	}

	log.Infof("Message [%v] with UUID [%v] is VALID.", tid, uuid)

	if isMessagePastPublishSLA(publishDate, appConfig.Threshold) {
		log.Infof("Message [%v] with UUID [%v] is past publish SLA, skipping.", tid, uuid)
		return false, nil
	}

	return true, &SchedulerParam{publishedContent, publishDate, tid, valRes.IsMarkedDeleted, metricContainer, environments}
}

func getValidationEndpointKey(publishedContent content.Content, tid string, uuid string) string {
	return publishedContent.GetType()
}

func isMessagePastPublishSLA(date time.Time, threshold int) bool {
	passedSLA := date.Add(time.Duration(threshold) * time.Second)
	return time.Now().After(passedSLA)
}
