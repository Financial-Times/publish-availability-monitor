package content

import (
	"net/http"

	uuidutils "github.com/Financial-Times/uuid-utils-go"
	log "github.com/Sirupsen/logrus"
)

type GenericContent struct {
	UUID          string `json:"uuid"`
	Type          string `json:"-"` //This field is for internal application usage
	BinaryContent []byte `json:"-"` //This field is for internal application usage
}

func (gc GenericContent) Initialize(binaryContent []byte) Content {
	gc.BinaryContent = binaryContent
	return gc
}

func (gc GenericContent) Validate(externalValidationEndpoint string, txID string, username string, password string) ValidationResponse {
	if uuidutils.ValidateUUID(gc.GetUUID()) != nil {
		log.Warnf("Generic content UUID is invalid: [%s]", gc.GetUUID())
		return ValidationResponse{IsValid: false, IsMarkedDeleted: gc.isMarkedDeleted()}
	}

	param := validationParam{
		gc.BinaryContent,
		externalValidationEndpoint,
		username,
		password,
		txID,
		gc.GetUUID(),
		gc.GetType(),
		true,
	}

	return doExternalValidation(
		param,
		gc.isValid,
		gc.isMarkedDeleted,
	)
}

func (gc GenericContent) GetType() string {
	return gc.Type
}

func (gc GenericContent) GetUUID() string {
	return gc.UUID
}

func (gc GenericContent) isValid(status int) bool {
	return status == http.StatusOK
}

func (gc GenericContent) isMarkedDeleted(status ...int) bool {
	return false
}
