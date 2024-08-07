package content

import (
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
	uuidutils "github.com/Financial-Times/uuid-utils-go"
)

type GenericContent struct {
	UUID          string   `json:"uuid"`
	EditorialDesk string   `json:"editorialDesk,omitempty"`
	Publication   []string `json:"publication,omitempty"`
	Type          string   `json:"-"` // This field is for internal application usage
	BinaryContent []byte   `json:"-"` // This field is for internal application usage
	Deleted       bool     `json:"deleted,omitempty"`
}

func (gc GenericContent) Initialize(binaryContent []byte) Content {
	gc.BinaryContent = binaryContent
	return gc
}

func (gc GenericContent) Validate(
	externalValidationEndpoint, tid, username, password string,
	log *logger.UPPLogger,
) ValidationResponse {
	if uuidutils.ValidateUUID(gc.GetUUID()) != nil {
		log.WithUUID(gc.GetUUID()).Warn("Generic content UUID is invalid")
		return ValidationResponse{IsValid: false, IsMarkedDeleted: gc.isMarkedDeleted()}
	}

	param := validationParam{
		binaryContent:    gc.BinaryContent,
		validationURL:    externalValidationEndpoint,
		username:         username,
		password:         password,
		tid:              tid,
		uuid:             gc.GetUUID(),
		contentType:      gc.GetType(),
		isGenericPublish: true,
	}

	return doExternalValidation(
		param,
		gc.isValid,
		gc.isMarkedDeleted,
		log,
	)
}

func (gc GenericContent) GetType() string {
	return gc.Type
}

func (gc GenericContent) GetUUID() string {
	return gc.UUID
}

func (gc GenericContent) GetEditorialDesk() string {
	return gc.EditorialDesk
}

func (gc GenericContent) GetPublication() []string {
	return gc.Publication
}

func (gc GenericContent) isValid(status int) bool {
	return status == http.StatusOK
}

func (gc GenericContent) isMarkedDeleted(status ...int) bool {
	return gc.Deleted
}
