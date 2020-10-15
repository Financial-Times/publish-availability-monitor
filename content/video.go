package content

import (
	"net/http"

	uuidutils "github.com/Financial-Times/uuid-utils-go"
	log "github.com/sirupsen/logrus"
)

const videoType = "video"

type Video struct {
	ID            string `json:"id"`
	Deleted       bool   `json:"deleted,omitempty"`
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
	return video.Deleted
}

func (video Video) GetType() string {
	return videoType
}

func (video Video) GetUUID() string {
	return video.ID
}
