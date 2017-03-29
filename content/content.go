package content

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"encoding/xml"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/checks"
)

// Content is the interface for different type of contents from different CMSs.
type Content interface {
	Validate(externalValidationEndpoint string, txID string, username string, password string) ValidationResponse
	GetType() string
	GetUUID() string
}

type ValidationResponse struct {
	IsValid bool
	IsMarkedDeleted bool
}

const systemIDKey = "Origin-System-Id"

const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

var infoLogger *log.Logger
var warnLogger *log.Logger
var errorLogger *log.Logger
var httpCaller checks.HttpCaller

func init() {
	//to be used for INFO-level logging: info.Println("foo is now bar")
	infoLogger = log.New(os.Stdout, "INFO  - ", logPattern)
	//to be used for WARN-level logging: warn.Println("foo is now bar")
	warnLogger = log.New(os.Stdout, "WARN  - ", logPattern)
	//to be used for ERROR-leve logging: errorL.Println("foo is now bar")
	errorLogger = log.New(os.Stdout, "ERROR - ", logPattern)
	httpCaller = checks.NewHttpCaller(10)
}

// UnmarshalContent unmarshals the message body into the appropriate content type based on the systemID header.
func UnmarshalContent(msg consumer.Message) (Content, error) {
	headers := msg.Headers

	systemID := headers[systemIDKey]
	switch systemID {
	case "http://cmdb.ft.com/systems/methode-web-pub":
		var eomFile EomFile

		err := json.Unmarshal([]byte(msg.Body), &eomFile)
		if err != nil {
			return nil, err
		}
		xml.Unmarshal([]byte(eomFile.Attributes), &eomFile.Source)
		eomFile = eomFile.initType()
		return eomFile, err
	case "http://cmdb.ft.com/systems/wordpress":
		var wordPressMsg WordPressMessage
		err := json.Unmarshal([]byte(msg.Body), &wordPressMsg)
		return wordPressMsg, err
	case "http://cmdb.ft.com/systems/brightcove":
		var video Video
		err := json.Unmarshal([]byte(msg.Body), &video)
		return video, err
	default:
		return nil, fmt.Errorf("Unsupported content with system ID: [%s].", systemID)
	}
}
