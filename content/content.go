package content

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Financial-Times/publish-availability-monitor/v2/checks"
	log "github.com/sirupsen/logrus"
)

// Content is the interface for different type of contents from different CMSs.
type Content interface {
	Initialize(binaryContent []byte) Content
	Validate(externalValidationEndpoint string, txID string, username string, password string) ValidationResponse
	GetType() string
	GetUUID() string
}

type ValidationResponse struct {
	IsValid         bool
	IsMarkedDeleted bool
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

var httpCaller checks.HttpCaller

func init() {
	httpCaller = checks.NewHttpCaller(10)
}

func doExternalValidation(p validationParam, validCheck func(int) bool, deletedCheck func(...int) bool) ValidationResponse {
	if p.validationURL == "" {
		log.Warnf("External validation for content uuid=[%s] transaction_id=[%s]. Validation endpoint URL is missing for content type=[%s]", p.uuid, p.txID, p.contentType)
		return ValidationResponse{false, deletedCheck()}
	}

	resp, err := httpCaller.DoCall(checks.Config{
		HttpMethod: "POST", Url: p.validationURL, Username: p.username, Password: p.password,
		TxId:        checks.ConstructPamTxId(p.txID),
		ContentType: "application/json", Entity: bytes.NewReader(p.binaryContent)})

	if err != nil {
		log.Warnf("External validation for content uuid=[%s] transaction_id=[%s] validationURL=[%s], creating validation request error: [%v]. Skipping external validation.", p.uuid, p.txID, p.validationURL, err)
		return ValidationResponse{true, deletedCheck()}
	}
	defer cleanupResp(resp)

	log.Infof("External validation for content uuid=[%s] transaction_id=[%s] validationURL=[%s], received statusCode [%d]", p.uuid, p.txID, p.validationURL, resp.StatusCode)

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("External validation for content uuid=[%s] transaction_id=[%s] validationURL=[%s], reading response body error: [%v]", p.uuid, p.txID, p.validationURL, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		log.Infof("External validation for content uuid=[%s] transaction_id=[%s] validationURL=[%s], received statusCode [%d], received error: [%v]", p.uuid, p.txID, p.validationURL, resp.StatusCode, string(bs))
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Infof("External validation for content uuid=[%s] transaction_id=[%s] validationURL=[%s], received statusCode [%d], content is marked as deleted.", p.uuid, p.txID, p.validationURL, resp.StatusCode)
	}

	return ValidationResponse{validCheck(resp.StatusCode), deletedCheck(resp.StatusCode)}
}

func cleanupResp(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		log.Warnf("External validation cleanup failed with error: [%v]", err)
	}
	err = resp.Body.Close()
	if err != nil {
		log.Warnf("External validation cleanup failed with error: [%v]", err)
	}
}
