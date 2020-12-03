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

// Attributes is the data structure that models methode content placeholders attributes
type Attributes struct {
	XMLName             xml.Name `xml:"ObjectMetadata"`
	SourceCode          string   `xml:"EditorialNotes>Sources>Source>SourceCode"`
	LastPublicationDate string   `xml:"OutputChannels>DIFTcom>DIFTcomLastPublication"`
	RefField            string   `xml:"WiresIndexing>ref_field"`
	ServiceId           string   `xml:"WiresIndexing>serviceid"`
	Category            string   `xml:"WiresIndexing>category"`
	IsDeleted           bool     `xml:"OutputChannels>DIFTcom>DIFTcomMarkDeleted"`
	OriginalUUID        string   `xml:"EditorialNotes>OriginalUUID"`
}

func (eomfile EomFile) Initialize(binaryContent []byte) Content {
	eomfile.BinaryContent = binaryContent
	return eomfile
}

func (eomfile EomFile) Validate(externalValidationEndpoint string, txID string, username string, password string) ValidationResponse {
	validationParam := validationParam{
		eomfile.BinaryContent,
		externalValidationEndpoint,
		username,
		password,
		txID,
		eomfile.GetUUID(),
		eomfile.GetType(),
		false,
	}

	return doExternalValidation(
		validationParam,
		eomfile.isValid,
		eomfile.isMarkedDeleted,
	)
}

func (eomfile EomFile) isValid(status int) bool {
	return status == http.StatusOK || status == http.StatusNotFound
}

func (eomfile EomFile) isMarkedDeleted(status ...int) bool {
	return len(status) == 1 && status[0] == http.StatusNotFound
}

func (eomfile EomFile) GetType() string {
	return eomfile.Type
}

func (eomfile EomFile) GetUUID() string {
	return eomfile.UUID
}
