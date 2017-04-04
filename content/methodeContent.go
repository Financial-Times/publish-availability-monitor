package content

import (
	"encoding/xml"
	"net/http"
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
	BinaryContent    []byte        `json:"-"` //This field is for internal application usage
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

func (eomfile EomFile) Initialize(binaryContent []byte) Content {
	eomfile.BinaryContent = binaryContent
	return eomfile.initType()
}

func (eomfile EomFile) Validate(externalValidationEndpoint string, txID string, username string, password string) ValidationResponse {
	contentUUID := eomfile.UUID
	if !isUUIDValid(contentUUID) {
		warnLogger.Printf("Eomfile invalid: invalid UUID: [%s]. transaction_id=[%s]", contentUUID, txID)
		return ValidationResponse{IsValid:false}
	}

	validationParam := validationParam{
		eomfile.BinaryContent,
		externalValidationEndpoint,
		username,
		password,
		txID,
		eomfile.GetUUID(),
		eomfile.GetType(),
	}

	return doExternalValidation(
		validationParam,
		eomfile.isValid,
		eomfile.isMarkedDeleted,
	)
}

func (eomfile EomFile) isValid(status int) bool {
	if status == http.StatusTeapot {
		return false
	}

	//invalid  contentplaceholder (link file) will not be published so do not monitor
	if status == http.StatusUnprocessableEntity {
		return false
	}

	return true
}

func (eomfile EomFile) isMarkedDeleted(status ...int) bool {
	if eomfile.Type == "Image" || eomfile.Type == "EOM::WebContainer" {
		return false
	}

	if len(status) == 1 && status[0] == http.StatusNotFound {
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
