package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	log "github.com/Sirupsen/logrus"
)

const systemIDKey = "Origin-System-Id"

type MessageHandler interface {
	HandleMessage(msg consumer.Message)
}

func NewKafkaMessageHandler(typeRes content.TypeResolver, appConfig *config.AppConfig, environments *envs.ThreadSafeEnvironments, subscribedFeeds map[string][]feeds.Feed, metricSink chan metrics.PublishMetric, metricContainer *metrics.PublishMetricsHistory) MessageHandler {
	return &kafkaMessageHandler{
		typeRes:         typeRes,
		appConfig:       appConfig,
		environments:    environments,
		subscribedFeeds: subscribedFeeds,
		metricSink:      metricSink,
		metricContainer: metricContainer,
	}
}

type kafkaMessageHandler struct {
	typeRes         content.TypeResolver
	appConfig       *config.AppConfig
	environments    *envs.ThreadSafeEnvironments
	subscribedFeeds map[string][]feeds.Feed
	metricSink      chan metrics.PublishMetric
	metricContainer *metrics.PublishMetricsHistory
}

func (h *kafkaMessageHandler) HandleMessage(msg consumer.Message) {
	tid := msg.Headers["X-Request-Id"]
	log.Infof("Received message with TID [%v]", tid)

	if h.isIgnorableMessage(tid) {
		log.Infof("Message [%v] is ignorable. Skipping...", tid)
		return
	}

	publishDateString := msg.Headers["Message-Timestamp"]
	publishDate, err := time.Parse(checks.DateLayout, publishDateString)
	if err != nil {
		log.Errorf("Cannot parse publish date [%v] from message [%v], error: [%v]",
			publishDateString, tid, err.Error())
		return
	}

	publishedContent, err := h.unmarshalContent(msg)
	if err != nil {
		log.Warnf("Cannot unmarshal message [%v], error: [%v]", tid, err.Error())
		return
	}

	var paramsToSchedule []*checks.SchedulerParam

	for _, preCheck := range checks.МainPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate, h.appConfig, h.metricContainer, h.environments)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		} else {
			//if a main check is not ok, additional checks make no sense
			return
		}
	}

	for _, preCheck := range checks.AdditionalPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate, h.appConfig, h.metricContainer, h.environments)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		}
	}

	hC := httpcaller.NewCaller(10)

	//key is the endpoint alias from the config
	endpointSpecificChecks := map[string]checks.EndpointSpecificCheck{
		"content":                 checks.NewContentCheck(hC),
		"content-neo4j":           checks.NewContentNeo4jCheck(hC),
		"complementary-content":   checks.NewContentCheck(hC),
		"internal-components":     checks.NewContentCheck(hC),
		"S3":                      checks.NewS3Check(hC),
		"enrichedContent":         checks.NewContentCheck(hC),
		"lists":                   checks.NewContentCheck(hC),
		"notifications":           checks.NewNotificationsCheck(hC, h.subscribedFeeds, "notifications"),
		"notifications-push":      checks.NewNotificationsCheck(hC, h.subscribedFeeds, "notifications-push"),
		"list-notifications":      checks.NewNotificationsCheck(hC, h.subscribedFeeds, "list-notifications"),
		"list-notifications-push": checks.NewNotificationsCheck(hC, h.subscribedFeeds, "list-notifications-push"),
	}

	for _, scheduleParam := range paramsToSchedule {
		checks.ScheduleChecks(scheduleParam, h.subscribedFeeds, endpointSpecificChecks, h.appConfig, h.metricSink)
	}
}

func (h *kafkaMessageHandler) isIgnorableMessage(tid string) bool {
	return h.isSyntheticTransactionID(tid) || h.isContentCarouselTransactionID(tid)
}

func (h *kafkaMessageHandler) isSyntheticTransactionID(tid string) bool {
	return strings.HasPrefix(tid, "SYNTHETIC")
}

func (h *kafkaMessageHandler) isContentCarouselTransactionID(tid string) bool {
	return carouselTransactionIDRegExp.MatchString(tid)
}

// UnmarshalContent unmarshals the message body into the appropriate content type based on the systemID header.
func (h *kafkaMessageHandler) unmarshalContent(msg consumer.Message) (content.Content, error) {
	binaryContent := []byte(msg.Body)

	headers := msg.Headers
	systemID := headers[systemIDKey]
	txID := msg.Headers["X-Request-Id"]
	switch systemID {
	case "http://cmdb.ft.com/systems/methode-web-pub":
		var eomFile content.EomFile

		err := json.Unmarshal(binaryContent, &eomFile)
		if err != nil {
			return nil, err
		}
		xml.Unmarshal([]byte(eomFile.Attributes), &eomFile.Source)
		eomFile = eomFile.Initialize(binaryContent).(content.EomFile)
		theType, resolvedUUID, err := h.typeRes.ResolveTypeAndUUID(eomFile, txID)
		if err != nil {
			return nil, fmt.Errorf("couldn't map kafka message to methode Content while fetching its type and uuid. %v", err)
		}
		eomFile.Type = theType
		eomFile.UUID = resolvedUUID
		return eomFile, nil
	case "http://cmdb.ft.com/systems/wordpress":
		var wordPressMsg content.WordPressMessage
		err := json.Unmarshal(binaryContent, &wordPressMsg)
		if err != nil {
			return nil, err
		}
		return wordPressMsg.Initialize(binaryContent), nil
	case "http://cmdb.ft.com/systems/next-video-editor":
		var video content.Video
		err := json.Unmarshal(binaryContent, &video)
		if err != nil {
			return nil, err
		}
		return video.Initialize(binaryContent), nil
	default:
		return nil, fmt.Errorf("unsupported content with system ID: [%s]", systemID)
	}
}
