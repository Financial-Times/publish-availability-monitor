package content

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"launchpad.net/xmlpath"
)

const image = "Image"
const webContainer = "EOM::WebContainer"
const compoundStory = "EOM::CompoundStory"
const story = "EOM::Story"

const titleXPath = "/doc/lead/lead-headline/headline/ln"
const channelXPath = "/props/productInfo/name"
const webTypeXPath = "//ObjectMetadata/FTcom/DIFTcomWebType"
const filePathXPath = "//ObjectMetadata/EditorialNotes/ObjectLocation"
const sourceXPath = "//ObjectMetadata/EditorialNotes/Sources/Source/SourceCode"
const markDeletedFlagXPath = "//ObjectMetadata/OutputChannels/DIFTcom/DIFTcomMarkDeleted"

const expectedWebChannel = "FTcom"
const expectedWebTypePrefix = "digitalList"
const expectedFilePathSuffix = ".xml"
const expectedSourceCode = "FT"

// EomFile models Methode content
type EomFile struct {
	UUID             string `json:"uuid"`
	Type             string `json:"type"`
	Value            string `json:"value"`
	Attributes       string `json:"attributes"`
	SystemAttributes string `json:"systemAttributes"`
	UsageTickets     string `json:"usageTickets"`
	WorkflowStatus   string `json:"workflowStatus"`
}

func (eomfile EomFile) IsValid(externalValidationEndpoint string) bool {
	contentUUID := eomfile.UUID
	if !isUUIDValid(contentUUID) {
		warnLogger.Printf("Eomfile invalid: invalid UUID: [%s]", contentUUID)
		return false
	}

	contentType := eomfile.Type
	switch contentType {
	case webContainer:
		return isListValid(eomfile)
	case compoundStory:
		return isCompoundStoryValid(eomfile) && isExternalValidationSuccessful(eomfile, externalValidationEndpoint)
	case story:
		return isStoryValid(eomfile) && isExternalValidationSuccessful(eomfile, externalValidationEndpoint)
	case image:
		return isImageValid(eomfile)
	default:
		warnLogger.Printf("Eomfile invalid: unexpected content type: [%s]", contentType)
		return false
	}
}

func (eomfile EomFile) IsMarkedDeleted() bool {
	if eomfile.Type == "Image" || eomfile.Type == "EOM::WebContainer" {
		return false
	}
	markDeletedFlag, ok := getXPathValue(eomfile.Attributes, eomfile, markDeletedFlagXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", markDeletedFlagXPath)
		return false
	}
	infoLogger.Printf("MarkAsDeletedFlag: [%v]", markDeletedFlag)
	if markDeletedFlag == "True" {
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

func isListValid(eomfile EomFile) bool {
	webType, ok := getXPathValue(eomfile.Attributes, eomfile, webTypeXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", webTypeXPath)
		return false
	}
	if strings.HasPrefix(webType, expectedWebTypePrefix) {
		return true
	}
	return false
}

func isCompoundStoryValid(eomfile EomFile) bool {
	return isSupportedFileType(eomfile) &&
		isWebChannel(eomfile) &&
		hasTitle(eomfile) &&
		isSupportedCompoundStorySourceCode(eomfile)
}

func isStoryValid(eomfile EomFile) bool {
	return isSupportedFileType(eomfile) &&
		isWebChannel(eomfile) &&
		hasTitle(eomfile) &&
		isSupportedStorySourceCode(eomfile)
}

func isSupportedFileType(eomfile EomFile) bool {
	filePath, ok := getXPathValue(eomfile.Attributes, eomfile, filePathXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", filePathXPath)
		return false
	}
	if strings.HasSuffix(filePath, expectedFilePathSuffix) {
		return true
	}
	return false
}

func isWebChannel(eomfile EomFile) bool {
	channel, ok := getXPathValue(eomfile.SystemAttributes, eomfile, channelXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", channelXPath)
		return false
	}
	if channel == expectedWebChannel {
		return true
	}
	return false
}

func hasTitle(eomfile EomFile) bool {
	if len(eomfile.Value) == 0 {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(eomfile.Value)
	if err != nil {
		warnLogger.Printf("Cannot decode Base64-encoded eomfile value: [%v]", err.Error())
		return false
	}
	articleXML := string(decoded[:])

	title, ok := getXPathValue(articleXML, eomfile, titleXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", titleXPath)
		return false
	}

	title = strings.TrimSpace(title)
	if len(title) > 0 {
		return true
	}
	warnLogger.Println("Title length is 0")
	return false
}

func isImageValid(eomfile EomFile) bool {
	if len(eomfile.Value) == 0 {
		warnLogger.Println("Image content length is 0")
		return false
	}
	return true
}

func isSupportedCompoundStorySourceCode(eomfile EomFile) bool {
	sourceCode, ok := getXPathValue(eomfile.Attributes, eomfile, sourceXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", sourceXPath)
		return false
	}
	if sourceCode == expectedSourceCode {
		return true
	}
	return false
}

func getXPathValue(xml string, eomfile EomFile, lookupPath string) (string, bool) {
	path := xmlpath.MustCompile(lookupPath)
	root, err := xmlpath.Parse(strings.NewReader(xml))
	if err != nil {
		warnLogger.Printf("Cannot parse XML of eomfile using xpath [%v], error: [%v]", err.Error(), lookupPath)
		return "", false
	}
	xpathValue, ok := path.String(root)
	return xpathValue, ok

}

func isSupportedStorySourceCode(eomfile EomFile) bool {
	validSourceCodes := [1]string{"FT"}

	sourceCode, ok := getXPathValue(eomfile.Attributes, eomfile, sourceXPath)
	if !ok {
		warnLogger.Printf("Cannot match node in XML using xpath [%v]", sourceXPath)
		return false
	}
	for _, expected := range validSourceCodes {
		if sourceCode == expected {
			return true
		}
	}
	return false
}

func isExternalValidationSuccessful(eomfile EomFile, validationURL string) bool {
	if validationURL == "" {
		warnLogger.Printf("External validation for content uuid=[%s]. Validation endpoint URL is missing for content type=[%s]. Skipping external validation.", eomfile.UUID, eomfile.Type)
		return true
	}
	marshalled, err := json.Marshal(eomfile)
	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] error: [%v]. Skipping external validation.", eomfile.UUID, err)
		return true
	}
	resp, err := http.Post(validationURL+"/"+eomfile.UUID, "application/json", bytes.NewBuffer(marshalled))
	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] error: [%v]. Skipping external validation.", eomfile.UUID, err)
		return true
	}
	defer cleanupResp(resp)

	infoLogger.Printf("External validation for content uuid=[%s] received statusCode: [%d]", eomfile.UUID, resp.StatusCode)

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		warnLogger.Printf("External validation for content uuid=[%s] reading response body error: [%v]. ", eomfile.UUID, err)
	} else {
		infoLogger.Printf("External validation for content uuid=[%s] received response body: [%v]", eomfile.UUID, string(bs))
	}
	if resp.StatusCode > 404 {
		return false
	}
	return true
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
