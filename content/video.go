package content

import (
	"regexp"

	log "github.com/Sirupsen/logrus"
)

const videoType = "video"

var idRegexp, _ = regexp.Compile("^\\d+$")

type Video struct {
	UUID        string `json:"uuid"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	UpdatedAt   string `json:"updated_at"`
	PublishedAt string `json:"published_at"`
}

func (v Video) IsValid(externalValidationEndpoint string, txId string, username string, password string) bool {
	contentUUID := v.UUID
	if !isUUIDValid(contentUUID) {
		log.Warnf("Video invalid: invalid UUID: [%s]", contentUUID)
		return false
	}
	if !idRegexp.MatchString(v.Id) {
		log.Warnf("Video invalid: invalid ID: [%s]", v.Id)
		return false
	}
	return true
}

func (v Video) IsMarkedDeleted() bool {
	if v.PublishedAt != "" || v.UpdatedAt != "" {
		return false
	}
	return true
}

func (v Video) GetType() string {
	return videoType
}

func (v Video) GetUUID() string {
	return v.UUID
}
