package content

import (
	"net/http"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/Financial-Times/uuid-utils-go"
)

const videoType = "video"

var idRegexp, _ = regexp.Compile("^\\d+$")

type Video struct {
	UUID          string `json:"uuid"`
	Id            string `json:"id"`
	Name          string `json:"name"`
	UpdatedAt     string `json:"updated_at"`
	PublishedAt   string `json:"published_at"`
	BinaryContent []byte `json:"-"` //This field is for internal application usage
}

func (video Video) Initialize(binaryContent []byte) Content {
	video.BinaryContent = binaryContent
	return video
}

func (video Video) Validate(externalValidationEndpoint string, txId string, username string, password string) ValidationResponse {
	if uuidutils.ValidateUUID(video.GetUUID()) != nil {
		log.Warnf("Video invalid: invalid UUID: [%s]", video.GetUUID())
		return ValidationResponse{IsValid: false, IsMarkedDeleted: video.isMarkedDeleted()}
	}

	if !idRegexp.MatchString(video.Id) {
		log.Warnf("Video invalid: invalid ID: [%s]", video.Id)
		return ValidationResponse{IsValid: false, IsMarkedDeleted: video.isMarkedDeleted()}
	}

	validationParam := validationParam{
		video.BinaryContent,
		externalValidationEndpoint,
		username,
		password,
		txId,
		video.GetUUID(),
		video.GetType(),
	}

	return doExternalValidation(
		validationParam,
		video.isValid,
		video.isMarkedDeleted,
	)
}

func (video Video) isValid(status int) bool {
	return status != http.StatusBadRequest
}

func (video Video) isMarkedDeleted(status ...int) bool {
	return video.PublishedAt == "" && video.UpdatedAt == ""
}

func (video Video) GetType() string {
	return videoType
}

func (video Video) GetUUID() string {
	return video.UUID
}
