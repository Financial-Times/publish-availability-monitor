package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	consumer "github.com/Financial-Times/message-queue-gonsumer"
	"github.com/Financial-Times/publish-availability-monitor/v2/content"
	log "github.com/sirupsen/logrus"
)

const systemIDKey = "Origin-System-Id"

type MessageHandler interface {
	HandleMessage(msg consumer.Message)
}

func NewKafkaMessageHandler(typeRes typeResolver) MessageHandler {
	return &kafkaMessageHandler{typeRes: typeRes}
}

type kafkaMessageHandler struct {
	typeRes typeResolver
}

func (h *kafkaMessageHandler) HandleMessage(msg consumer.Message) {
	tid := msg.Headers["X-Request-Id"]
	log.Infof("Received message with TID [%v]", tid)

	if h.isIgnorableMessage(tid) {
		log.Infof("Message [%v] is ignorable. Skipping...", tid)
		return
	}

	publishDateString := msg.Headers["Message-Timestamp"]
	publishDate, err := time.Parse(dateLayout, publishDateString)
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

	var paramsToSchedule []*schedulerParam

	for _, preCheck := range mainPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		} else {
			//if a main check is not ok, additional checks make no sense
			return
		}
	}

	for _, preCheck := range additionalPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		}
	}

	for _, scheduleParam := range paramsToSchedule {
		scheduleChecks(scheduleParam)
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
		theType, resolvedUuid, err := h.typeRes.ResolveTypeAndUuid(eomFile, txID)
		if err != nil {
			return nil, fmt.Errorf("couldn't map kafka message to methode Content while fetching its type and uuid. %v", err)
		}
		eomFile.Type = theType
		eomFile.UUID = resolvedUuid
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
