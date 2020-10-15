package main

import (
	"strings"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/content"
	uuidutils "github.com/Financial-Times/uuid-utils-go"
	log "github.com/sirupsen/logrus"
)

var uuidDeriver = uuidutils.NewUUIDDeriverWith(uuidutils.IMAGE_SET)

func mainPreChecks() []func(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam) {
	return []func(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam){
		mainPreCheck,
	}
}

func additionalPreChecks() []func(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam) {
	return []func(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam){
		imagePreCheck,
		internalComponentsPreCheck,
	}
}

func mainPreCheck(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam) {
	uuid := publishedContent.GetUUID()
	validationEndpointKey := getValidationEndpointKey(publishedContent, tid, uuid)
	var validationEndpoint string
	var found bool
	var username string
	var password string

	if validationEndpoint, found = appConfig.ValidationEndpoints[validationEndpointKey]; found {
		username, password = getValidationCredentials()
	}

	valRes := publishedContent.Validate(validationEndpoint, tid, username, password)
	if !valRes.IsValid {
		log.Infof("Message [%v] with UUID [%v] is INVALID, skipping...", tid, uuid)
		return false, nil
	}

	log.Infof("Message [%v] with UUID [%v] is VALID.", tid, uuid)

	if isMessagePastPublishSLA(publishDate, appConfig.Threshold) {
		log.Infof("Message [%v] with UUID [%v] is past publish SLA, skipping.", tid, uuid)
		return false, nil
	}

	return true, &schedulerParam{publishedContent, publishDate, tid, valRes.IsMarkedDeleted, &metricContainer, environments}
}

// for images we need to check their corresponding image sets
// the image sets don't have messages of their own so we need to create one
func imagePreCheck(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam) {
	if publishedContent.GetType() != "Image" {
		return false, nil
	}

	eomFile, ok := publishedContent.(content.EomFile)
	if !ok {
		log.Errorf("Cannot assert that message [%v] with UUID [%v] and type 'Image' is an EomFile.", tid, publishedContent.GetUUID())
		return false, nil
	}

	imageSetEomFile := spawnImageSet(eomFile)
	if imageSetEomFile.UUID == "" {
		return false, nil
	}

	return true, &schedulerParam{imageSetEomFile, publishDate, tid, false, &metricContainer, environments}
}

// if this is normal content, schedule checks for internal components also
func internalComponentsPreCheck(publishedContent content.Content, tid string, publishDate time.Time) (bool, *schedulerParam) {
	if publishedContent.GetType() != "EOM::CompoundStory" {
		return false, nil
	}

	eomFileForInternalComponentsCheck, ok := publishedContent.(content.EomFile)
	if !ok {
		log.Errorf("Cannot assert that message [%v] with UUID [%v] and type 'EOM::CompoundStory' is an EomFile.", tid, publishedContent.GetUUID())
		return false, nil
	}
	eomFileForInternalComponentsCheck.Type = "InternalComponents"

	var internalComponentsValidationEndpoint = appConfig.ValidationEndpoints["InternalComponents"]
	var usr, pass = getValidationCredentials()

	icValRes := publishedContent.Validate(internalComponentsValidationEndpoint, tid, usr, pass)
	if !icValRes.IsValid {
		log.Infof("Message [%v] with UUID [%v] has INVALID internal components, skipping internal components schedule check.", tid, publishedContent.GetUUID())
		return false, nil
	}

	return true, &schedulerParam{eomFileForInternalComponentsCheck, publishDate, tid, icValRes.IsMarkedDeleted, &metricContainer, environments}
}

func getValidationCredentials() (string, string) {
	if strings.Contains(validatorCredentials, ":") {
		unpw := strings.SplitN(validatorCredentials, ":", 2)
		return unpw[0], unpw[1]
	}

	return "", ""
}

func getValidationEndpointKey(publishedContent content.Content, tid string, uuid string) string {
	validationEndpointKey := publishedContent.GetType()
	if strings.Contains(publishedContent.GetType(), "EOM::CompoundStory") {
		_, ok := publishedContent.(content.EomFile)
		if !ok {
			log.Errorf("Cannot assert that message [%v] with UUID [%v] and type 'EOM::CompoundStory' is an EomFile.", tid, uuid)
			return ""
		}

	}
	return validationEndpointKey
}

func isMessagePastPublishSLA(date time.Time, threshold int) bool {
	passedSLA := date.Add(time.Duration(threshold) * time.Second)
	return time.Now().After(passedSLA)
}

func spawnImageSet(imageEomFile content.EomFile) content.EomFile {
	imageSetEomFile := imageEomFile
	imageSetEomFile.Type = "ImageSet"

	imageUUID, err := uuidutils.NewUUIDFromString(imageEomFile.UUID)
	if err != nil {
		log.Warnf("Cannot generate UUID from image UUID string [%v]: [%v], skipping image set check.",
			imageEomFile.UUID, err.Error())
		return content.EomFile{}
	}

	imageSetUUID, err := uuidDeriver.From(imageUUID)
	if err != nil {
		log.Warnf("Cannot generate image set UUID: [%v], skipping image set check",
			err.Error())
		return content.EomFile{}
	}

	imageSetEomFile.UUID = imageSetUUID.String()
	return imageSetEomFile
}
