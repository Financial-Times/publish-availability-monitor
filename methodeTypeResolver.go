package main

import (
	"encoding/xml"
	"fmt"

	"github.com/Financial-Times/publish-availability-monitor/v2/checks"
	"github.com/Financial-Times/publish-availability-monitor/v2/content"
	log "github.com/sirupsen/logrus"
)

var blogCategories = []string{"blog", "webchat-live-blogs", "webchat-live-qa", "webchat-markets-live", "fastft"}

type typeResolver interface {
	ResolveTypeAndUuid(eomFile content.EomFile, txID string) (string, string, error)
}

type methodeTypeResolver struct {
	resolver checks.UUIDResolver
}

func NewMethodeTypeResolver(uuidResolver checks.UUIDResolver) *methodeTypeResolver {
	return &methodeTypeResolver{
		resolver: uuidResolver,
	}
}

func (m *methodeTypeResolver) ResolveTypeAndUuid(eomFile content.EomFile, txID string) (string, string, error) {
	contentType := eomFile.ContentType
	contentSrc := eomFile.Source.SourceCode
	if contentSrc == "ContentPlaceholder" && contentType == "EOM::CompoundStory" {
		resolvedUUID, err := m.resolveUUID(eomFile, txID)
		if err != nil {
			return "", "", err
		}

		theType := "EOM::CompoundStory_External_CPH"
		cphUUID := eomFile.UUID
		if resolvedUUID != "" {
			theType = "EOM::CompoundStory_Internal_CPH"
			cphUUID = resolvedUUID
		}
		log.Infof("For placeholder resolved tid=%v type=%v uuid=%v", txID, theType, cphUUID)
		return theType, cphUUID, nil
	}

	if contentSrc == "DynamicContent" && contentType == "EOM::CompoundStory" {
		return "EOM::CompoundStory_DynamicContent", eomFile.UUID, nil
	}

	return eomFile.ContentType, eomFile.UUID, nil
}

func (m *methodeTypeResolver) resolveUUID(eomFile content.EomFile, txID string) (string, error) {
	attributes, err := m.buildAttributes(eomFile.Attributes)
	if err != nil {
		return "", err
	}

	var uuid string
	if attributes.OriginalUUID != "" {
		uuid, err = m.resolver.ResolveOriginalUUID(attributes.OriginalUUID, txID)
		if err != nil {
			return "", err
		}

		if uuid == "" {
			return "", fmt.Errorf("couldn't resolve CPH uuid for tid=%v, OriginalUUID=%v is not present in the database", txID, attributes.OriginalUUID)
		}
	} else {
		if isBlogCategory(attributes) {
			uuid, err = m.resolver.ResolveIdentifier(attributes.ServiceId, attributes.RefField, txID)
			if err != nil {
				return "", fmt.Errorf("couldn't resolve blog uuid, error was: %v", err)
			}
		}
	}

	return uuid, nil
}

func (m *methodeTypeResolver) buildAttributes(attributesXML string) (content.Attributes, error) {
	var attrs content.Attributes
	if err := xml.Unmarshal([]byte(attributesXML), &attrs); err != nil {
		return content.Attributes{}, err
	}
	return attrs, nil
}

func isBlogCategory(attributes content.Attributes) bool {
	for _, c := range blogCategories {
		if c == attributes.Category {
			return true
		}
	}
	return false
}
