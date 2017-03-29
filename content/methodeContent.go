package content

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"encoding/xml"

	"github.com/Financial-Times/publish-availability-monitor/checks"
)

// EomFile models Methode content
type EomFile struct {
	UUID             string        `json:"uuid"`
	LinkedObjects    []interface{} `json:"linkedObjects"`
	ContentType      string        `json:"type"`
	Value            string        `json:"value"`
	Attributes       string        `json:"attributes"`
	SystemAttributes string        `json:"systemAttributes"`
	UsageTickets     string        `json:"usageTickets"`
	WorkflowStatus   string        `json:"workflowStatus"`
	Type             string        `json:"-"` //This field is for internal application usage
	Source           Source        `json:"-"` //This field is for internal application usage
}

type Source struct {
	XMLName    xml.Name `xml:"ObjectMetadata"`
	SourceCode string   `xml:"EditorialNotes>Sources>Source>SourceCode"`
}

func (eomfile EomFile) initType() EomFile {
	contentType := eomfile.ContentType
	contentSrc := eomfile.Source.SourceCode

	if contentSrc == "ContentPlaceholder" && contentType == "EOM::CompoundStory" {
		eomfile.Type = "EOM::CompoundStory_ContentPlaceholder"
		infoLogger.Printf("results [%v] ....", eomfile.Type)
		return eomfile
	}
	eomfile.Type = eomfile.ContentType
	return eomfile
}

func (eomfile EomFile) Validate(externalValidationEndpoint string, txID string, username string, password string) ValidationResponse {
	contentUUID := eomfile.UUID
	if !isUUIDValid(contentUUID) {
		warnLogger.Printf("Eomfile invalid: invalid UUID: [%s]. transaction_id=[%s]", contentUUID, txID)
		return ValidationResponse{IsValid:false}
	}

	isValid, statusCode := isExternalValidationSuccessful(eomfile, externalValidationEndpoint, txID, username, password)

	return ValidationResponse{IsValid:isValid, IsMarkedDeleted: eomfile.isMarkedDeleted(statusCode)}
}

func (eomfile EomFile) isMarkedDeleted(validationStatusCode int) bool {
	if eomfile.Type == "Image" || eomfile.Type == "EOM::WebContainer" {
		return false
	}

	if validationStatusCode == http.StatusNotFound {
		infoLogger.Printf("Eomfile with uuid=[%s] is marked as deleted!", eomfile.UUID)
		return true
	}

	return false
}

func (eomfile EomFile) GetType() string {
	return eomfile.Type
}

func (eomfile EomFile) GetUUID() string {
	return eomfile.UUID
}

func isExternalValidationSuccessful(eomfile EomFile, validationURL string, txID, username string, password string) (bool, int) {
	if validationURL == "" {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s]. Validation endpoint URL is missing for content type=[%s]", eomfile.UUID, txID, eomfile.Type)
		return false, 0
	}
	marshalled, err := json.Marshal(eomfile)
	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]. Skipping external validation.", eomfile.UUID, txID, err)
		return true, 0
	}

	resp, err := httpCaller.DoCallWithEntity(
		"POST", validationURL,
		username, password,
		checks.ConstructPamTxId(txID),
		"application/json", bytes.NewReader(marshalled))

	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]. Skipping external validation.", eomfile.UUID, txID, err)
		return true, 0
	}
	defer cleanupResp(resp)

	infoLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] received statusCode [%d]", eomfile.UUID, txID, resp.StatusCode)

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]", eomfile.UUID, txID, err)
	}

	if resp.StatusCode != http.StatusOK {
		infoLogger.Printf("External validation for content uuid=[%s] transaction_id=[%s] error: [%v]", eomfile.UUID, txID, string(bs))
	}

	// 422 invalid  contentplaceholder (link file) will not be published so do not monitor
	if resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusTeapot {
		return false, resp.StatusCode
	}

	return true, resp.StatusCode
}

func cleanupResp(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}
	err = resp.Body.Close()
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}
}
