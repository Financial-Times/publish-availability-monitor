package content

import (
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
	uuidutils "github.com/Financial-Times/uuid-utils-go"
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

func (video Video) Validate(externalValidationEndpoint, tid, username, password string, log *logger.UPPLogger) ValidationResponse {
	uuid := video.GetUUID()

	if uuidutils.ValidateUUID(uuid) != nil {
		log.WithUUID(uuid).Warnf("Invalid video UUID")
		return ValidationResponse{IsValid: false, IsMarkedDeleted: video.isMarkedDeleted()}
	}

	param := validationParam{
		binaryContent: video.BinaryContent,
		validationURL: externalValidationEndpoint,
		username:      username,
		password:      password,
		tid:           tid,
		uuid:          uuid,
		contentType:   video.GetType(),
	}

	return doExternalValidation(
		param,
		video.isValid,
		video.isMarkedDeleted,
		log,
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
