package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
)

const systemIDKey = "Origin-System-Id"

type MessageHandler interface {
	HandleMessage(msg kafka.FTMessage)
}

func NewKafkaMessageHandler(
	appConfig *config.AppConfig,
	environments *envs.Environments,
	subscribedFeeds map[string][]feeds.Feed,
	metricSink chan metrics.PublishMetric,
	metricContainer *metrics.History,
	e2eTestUUIDs []string,
	log *logger.UPPLogger,
) MessageHandler {
	return &kafkaMessageHandler{
		appConfig:       appConfig,
		environments:    environments,
		subscribedFeeds: subscribedFeeds,
		metricSink:      metricSink,
		metricContainer: metricContainer,
		e2eTestUUIDs:    e2eTestUUIDs,
		log:             log,
	}
}

type kafkaMessageHandler struct {
	appConfig       *config.AppConfig
	environments    *envs.Environments
	subscribedFeeds map[string][]feeds.Feed
	metricSink      chan metrics.PublishMetric
	metricContainer *metrics.History
	e2eTestUUIDs    []string
	log             *logger.UPPLogger
}

func (h *kafkaMessageHandler) HandleMessage(msg kafka.FTMessage) {
	tid := msg.Headers["X-Request-Id"]
	log := h.log.WithTransactionID(tid)

	log.Info("Received message")

	if h.isIgnorableMessage(msg) {
		log.Info("Message is ignorable. Skipping...")
		return
	}

	publishedContent, err := h.unmarshalContent(msg)
	if err != nil {
		h.log.WithError(err).Warn("Cannot unmarshal message")
		return
	}

	publishDateString := msg.Headers["Message-Timestamp"]
	publishDate, err := time.Parse(checks.DateLayout, publishDateString)
	if err != nil {
		h.log.WithError(err).Errorf("Cannot parse publish date [%v]",
			publishDateString)
		return
	}

	var paramsToSchedule []*checks.SchedulerParam

	for _, preCheck := range checks.MainPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate, h.appConfig, h.metricContainer, h.environments, h.log)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		} else {
			//if a main check is not ok, additional checks make no sense
			return
		}
	}

	hC := httpcaller.NewCaller(10)

	//key is the endpoint alias from the config
	endpointSpecificChecks := map[string]checks.EndpointSpecificCheck{
		"content":                  checks.NewContentCheck(hC),
		"content-neo4j":            checks.NewContentNeo4jCheck(hC),
		"content-collection-neo4j": checks.NewContentNeo4jCheck(hC),
		"complementary-content":    checks.NewContentCheck(hC),
		"internal-components":      checks.NewContentCheck(hC),
		"enrichedContent":          checks.NewContentCheck(hC),
		"lists":                    checks.NewContentCheck(hC),
		"pages":                    checks.NewContentCheck(hC),
		"notifications":            checks.NewNotificationsCheck(hC, h.subscribedFeeds, "notifications"),
		"notifications-push":       checks.NewNotificationsCheck(hC, h.subscribedFeeds, "notifications-push"),
		"list-notifications":       checks.NewNotificationsCheck(hC, h.subscribedFeeds, "list-notifications"),
		"list-notifications-push":  checks.NewNotificationsCheck(hC, h.subscribedFeeds, "list-notifications-push"),
		"page-notifications":       checks.NewNotificationsCheck(hC, h.subscribedFeeds, "page-notifications"),
		"page-notifications-push":  checks.NewNotificationsCheck(hC, h.subscribedFeeds, "page-notifications-push"),
	}

	for _, scheduleParam := range paramsToSchedule {
		checks.ScheduleChecks(scheduleParam, endpointSpecificChecks, h.appConfig, h.metricSink, h.e2eTestUUIDs, h.log)
	}
}

func (h *kafkaMessageHandler) isIgnorableMessage(msg kafka.FTMessage) bool {
	tid := msg.Headers["X-Request-Id"]

	isSynthetic := h.isSyntheticTransactionID(tid)
	isE2ETest := config.IsE2ETestTransactionID(tid, h.e2eTestUUIDs)
	isCarousel := h.isContentCarouselTransactionID(tid)

	if isSynthetic && isE2ETest {
		h.log.WithTransactionID(tid).Infof("Message is E2E Test.")
		return false
	}

	return isSynthetic || isCarousel
}

func (h *kafkaMessageHandler) isSyntheticTransactionID(tid string) bool {
	return strings.HasPrefix(tid, "SYNTHETIC")
}

func (h *kafkaMessageHandler) isContentCarouselTransactionID(tid string) bool {
	return carouselTransactionIDRegExp.MatchString(tid)
}

// UnmarshalContent unmarshals the message body into the appropriate content type based on the systemID header.
func (h *kafkaMessageHandler) unmarshalContent(msg kafka.FTMessage) (content.Content, error) {
	binaryContent := []byte(msg.Body)

	headers := msg.Headers
	systemID := headers[systemIDKey]
	switch systemID {
	case "http://cmdb.ft.com/systems/next-video-editor":
		if msg.Headers["Content-Type"] == "application/vnd.ft-upp-audio" {
			return unmarshalGenericContent(msg)
		}

		var video content.Video
		err := json.Unmarshal(binaryContent, &video)
		if err != nil {
			return nil, err
		}
		return video.Initialize(binaryContent), nil
	case "http://cmdb.ft.com/systems/cct", "http://cmdb.ft.com/systems/spark-lists", "http://cmdb.ft.com/systems/spark", "http://cmdb.ft.com/systems/spark-clips":
		return unmarshalGenericContent(msg)
	default:
		return nil, fmt.Errorf("unsupported content with system ID: [%s]", systemID)
	}
}

func unmarshalGenericContent(msg kafka.FTMessage) (content.GenericContent, error) {
	binaryContent := []byte(msg.Body)
	var genericContent content.GenericContent
	err := json.Unmarshal(binaryContent, &genericContent)
	if err != nil {
		return content.GenericContent{}, err
	}

	genericContent = genericContent.Initialize(binaryContent).(content.GenericContent)
	genericContent.Type = msg.Headers["Content-Type"]

	return genericContent, nil
}
