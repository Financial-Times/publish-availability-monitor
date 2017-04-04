package content

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bytes"
	"encoding/xml"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"io"
	"io/ioutil"
	"net/http"
)

// Content is the interface for different type of contents from different CMSs.
type Content interface {
	Initialize(binaryContent []byte) Content
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
var httpCaller checks.HttpCaller

func init() {
	//to be used for INFO-level logging: info.Println("foo is now bar")
	infoLogger = log.New(os.Stdout, "INFO  - ", logPattern)
	//to be used for WARN-level logging: warn.Println("foo is now bar")
	warnLogger = log.New(os.Stdout, "WARN  - ", logPattern)

	httpCaller = checks.NewHttpCaller(10)
}

// UnmarshalContent unmarshals the message body into the appropriate content type based on the systemID header.
func UnmarshalContent(msg consumer.Message) (Content, error) {
	binaryContent := []byte(msg.Body)

	headers := msg.Headers
	systemID := headers[systemIDKey]
	switch systemID {
	case "http://cmdb.ft.com/systems/methode-web-pub":
		var eomFile EomFile

		err := json.Unmarshal(binaryContent, &eomFile)
		if err != nil {
			return nil, err
		}
		xml.Unmarshal([]byte(eomFile.Attributes), &eomFile.Source)

		return eomFile.Initialize(binaryContent), err
	case "http://cmdb.ft.com/systems/wordpress":
		var wordPressMsg WordPressMessage
		err := json.Unmarshal(binaryContent, &wordPressMsg)
		return wordPressMsg.Initialize(binaryContent), err
	case "http://cmdb.ft.com/systems/brightcove":
		var video Video
		err := json.Unmarshal(binaryContent, &video)
		return video.Initialize(binaryContent), err
	default:
		return nil, fmt.Errorf("Unsupported content with system ID: [%s].", systemID)
	}
}

type validationParam struct {
	binaryContent []byte
	validationURL string
	username      string
	password      string
	txID          string
	uuid          string
	contentType   string
}

func doExternalValidation(p validationParam, validCheck func(int) bool, deletedCheck func(... int) bool) ValidationResponse {
	if p.validationURL == "" {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s]. Validation endpoint URL is missing for content type=[%s]", p.uuid, p.txID, p.contentType)
		return ValidationResponse{false, deletedCheck()}
	}

	resp, err := httpCaller.DoCallWithEntity(
		"POST", p.validationURL, p.username, p.password,
		checks.ConstructPamTxId(p.txID),
		"application/json", bytes.NewReader(p.binaryContent))

	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]. Skipping external validation.", p.uuid, p.txID, err)
		return ValidationResponse{true, deletedCheck()}
	}
	defer cleanupResp(resp)

	infoLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] received statusCode [%d]", p.uuid, p.txID, resp.StatusCode)

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]", p.uuid, p.txID, err)
	}

	if resp.StatusCode != http.StatusOK {
		infoLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]", p.uuid, p.txID, string(bs))
	}

	return ValidationResponse{validCheck(resp.StatusCode), deletedCheck(resp.StatusCode)}
}

func cleanupResp(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		warnLogger.Printf("External validation cleanup failed with error: [%v]", err)
	}
	err = resp.Body.Close()
	if err != nil {
		warnLogger.Printf("External validation cleanup failed with error: [%v]", err)
	}
}
