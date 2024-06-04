package content

import (
	"bytes"
	"io"
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
)

// Content is the interface for different type of contents from different CMSs.
type Content interface {
	Initialize(binaryContent []byte) Content
	Validate(
		externalValidationEndpoint, tid, username, password string,
		log *logger.UPPLogger,
	) ValidationResponse
	GetType() string
	GetUUID() string
	GetEditorialDesk() string
	GetPublication() []string
}

type ValidationResponse struct {
	IsValid         bool
	IsMarkedDeleted bool
}

type validationParam struct {
	binaryContent    []byte
	validationURL    string
	username         string
	password         string
	tid              string
	uuid             string
	contentType      string
	isGenericPublish bool
}

var httpCaller httpcaller.Caller

func init() {
	httpCaller = httpcaller.NewCaller(10)
}

func doExternalValidation(
	p validationParam,
	validCheck func(int) bool,
	deletedCheck func(...int) bool,
	log *logger.UPPLogger,
) ValidationResponse {
	logEntry := log.WithUUID(p.uuid).WithTransactionID(p.tid)

	if p.validationURL == "" {
		logEntry.Warnf(
			"External validation for content. Validation endpoint URL is missing for content type=[%s]",
			p.contentType,
		)
		return ValidationResponse{false, deletedCheck()}
	}

	logEntry = logEntry.WithField("validation_url", p.validationURL)

	contentType := "application/json"
	if p.isGenericPublish {
		contentType = p.contentType
	}

	resp, err := httpCaller.DoCall(httpcaller.Config{
		HTTPMethod:  "POST",
		URL:         p.validationURL,
		Username:    p.username,
		Password:    p.password,
		TID:         httpcaller.ConstructPamTID(p.tid),
		ContentType: contentType,
		Entity:      bytes.NewReader(p.binaryContent),
	})
	if err != nil {
		logEntry.WithError(err).
			Warn("External validation for content failed while creating validation request. Skipping external validation.")
		return ValidationResponse{true, deletedCheck()}
	}
	defer resp.Body.Close()

	logEntry.Infof("External validation received statusCode [%d]", resp.StatusCode)

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		logEntry.WithError(err).Warn("External validation reading response body error")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		logEntry.Infof(
			"External validation received statusCode [%d], received error: [%v]",
			resp.StatusCode,
			string(bs),
		)
	}

	if resp.StatusCode == http.StatusNotFound {
		logEntry.Infof(
			"External validation received statusCode [%d], content is marked as deleted.",
			resp.StatusCode,
		)
	}

	return ValidationResponse{validCheck(resp.StatusCode), deletedCheck(resp.StatusCode)}
}
