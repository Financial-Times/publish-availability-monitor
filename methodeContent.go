package main

import (
	"encoding/base64"
	"strings"

	"launchpad.net/xmlpath"
)

const image = "Image"
const webContainer = "EOM::WebContainer"
const compoundStory = "EOM::CompoundStory"

const titleXPath = "/doc/lead/lead-headline/headline/ln"
const channelXPath = "/props/productInfo/name"
const webTypeXPath = "//ObjectMetadata/FTcom/DIFTcomWebType"
const filePathXPath = "//ObjectMetadata/EditorialNotes/ObjectLocation"
const sourceXPath = "//ObjectMetadata/EditorialNotes/Sources/Source/SourceCode"

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
}

func (eomfile EomFile) isValid() bool {
	contentUUID := eomfile.UUID
	if !isUUIDValid(contentUUID) {
		warn.Printf("Eomfile invalid: invalid UUID: [%s]", contentUUID)
		return false
	}

	contentType := eomfile.Type
	switch contentType {
	case webContainer:
		return isListValid(eomfile)
	case compoundStory:
		return isCompoundStoryValid(eomfile)
	case image:
		return isImageValid(eomfile)
	default:
		warn.Printf("Eomfile invalid: unexpected content type: [%s]", contentType)
		return false
	}
}

func (eomfile EomFile) isMarkedDeleted() bool {
	if eomfile.Type == "Image" || eomfile.Type == "EOM::WebContainer" {
		return false
	}
	attributes := eomfile.Attributes
	markDeletedFlagXPath := "//ObjectMetadata/OutputChannels/DIFTcom/DIFTcomMarkDeleted"
	path := xmlpath.MustCompile(markDeletedFlagXPath)
	root, err := xmlpath.Parse(strings.NewReader(attributes))
	if err != nil {
		warn.Printf("Cannot parse attribute XML of eomFile, error: [%v]", err.Error())
		return false
	}
	markDeletedFlag, ok := path.String(root)
	if !ok {
		warn.Printf("Cannot match node in XML using xpath [%v]", markDeletedFlagXPath)
		return false
	}
	info.Printf("MarkAsDeletedFlag: [%v]", markDeletedFlag)
	if markDeletedFlag == "True" {
		return true
	}
	return false
}

func (eomfile EomFile) getType() string {
	return eomfile.Type
}

func (eomfile EomFile) getUUID() string {
	return eomfile.UUID
}

func isListValid(eomfile EomFile) bool {
	attributes := eomfile.Attributes
	path := xmlpath.MustCompile(webTypeXPath)
	root, err := xmlpath.Parse(strings.NewReader(attributes))
	if err != nil {
		warn.Printf("Cannot parse attribute XML of eomfile, error: [%v]", err.Error())
		return false
	}
	webType, ok := path.String(root)
	if !ok {
		warn.Printf("Cannot match node in XML using xpath [%v]", webTypeXPath)
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
		isSupportedSourceCode(eomfile)
}

func isSupportedFileType(eomfile EomFile) bool {
	attributes := eomfile.Attributes
	path := xmlpath.MustCompile(filePathXPath)
	root, err := xmlpath.Parse(strings.NewReader(attributes))
	if err != nil {
		warn.Printf("Cannot parse attribute XML of eomfile, error: [%v]", err.Error())
		return false
	}
	filePath, ok := path.String(root)
	if !ok {
		warn.Printf("Cannot match node in XML using xpath [%v]", filePathXPath)
		return false
	}
	if strings.HasSuffix(filePath, expectedFilePathSuffix) {
		return true
	}
	return false
}

func isWebChannel(eomfile EomFile) bool {
	systemAttributes := eomfile.SystemAttributes
	path := xmlpath.MustCompile(channelXPath)
	root, err := xmlpath.Parse(strings.NewReader(systemAttributes))
	if err != nil {
		warn.Printf("Cannot parse system attribute XML of eomfile, error: [%v]", err.Error())
		return false
	}
	channel, ok := path.String(root)
	if !ok {
		warn.Printf("Cannot match node in XML using xpath [%v]", channelXPath)
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
		warn.Printf("Cannot decode Base64-encoded eomfile value: [%v]", err.Error())
		return false
	}
	articleXML := string(decoded[:])

	root, err := xmlpath.Parse(strings.NewReader(articleXML))
	if err != nil {
		warn.Printf("Cannot parse value XML of eomfile, error: [%v]", err.Error())
		return false
	}

	path := xmlpath.MustCompile(titleXPath)
	title, ok := path.String(root)
	if !ok {
		warn.Printf("Cannot match node in XML using xpath [%v]", titleXPath)
		return false
	}

	title = strings.TrimSpace(title)
	if len(title) > 0 {
		return true
	}
	warn.Println("Title length is 0")
	return false
}

func isImageValid(eomfile EomFile) bool {
	if len(eomfile.Value) == 0 {
		warn.Println("Image content length is 0")
		return false
	}
	return true
}

func isSupportedSourceCode(eomfile EomFile) bool {
	attributes := eomfile.Attributes
	path := xmlpath.MustCompile(sourceXPath)
	root, err := xmlpath.Parse(strings.NewReader(attributes))
	if err != nil {
		warn.Printf("Cannot parse XML attribute of eomfile, error: [%v]", err.Error())
		return false
	}
	sourceCode, ok := path.String(root)
	if !ok {
		warn.Printf("Cannot match node in XML using xpath [%v]", sourceXPath)
		return false
	}
	if sourceCode == expectedSourceCode {
		return true
	}
	return false
}
