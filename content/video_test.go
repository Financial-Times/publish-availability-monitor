package content

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/stretchr/testify/assert"
)

func TestIsVideoValid_Valid(t *testing.T) {
	var videoValid = Video{
		ID:            "e28b12f7-9796-3331-b030-05082f0b8157",
		BinaryContent: []byte("valid-json"),
	}

	tid := "tid_1234"
	pamTID := httpcaller.ConstructPamTID(tid)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/map", req.RequestURI)
		assert.Equal(t, pamTID, req.Header.Get("X-Request-Id"))

		defer req.Body.Close()
		reqBody, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, videoValid.BinaryContent, reqBody)
	}))

	log := logger.NewUPPLogger("test", "PANIC")

	validationResponse := videoValid.Validate(testServer.URL+"/map", tid, "", "", log)
	assert.True(t, validationResponse.IsValid, "Video should be valid.")
}

func TestIsVideoValid_NoId(t *testing.T) {
	var videoNoID = Video{}
	log := logger.NewUPPLogger("test", "PANIC")

	validationResponse := videoNoID.Validate("", "", "", "", log)
	assert.False(t, validationResponse.IsValid, "Video should be invalid as it has no Id.")
}

func TestIsVideoValid_failedExternalValidation(t *testing.T) {
	var videoInvalid = Video{
		ID:            "e28b12f7-9796-3331-b030-05082f0b8157",
		BinaryContent: []byte("invalid-json"),
	}

	tid := "tid_1234"
	pamTID := httpcaller.ConstructPamTID(tid)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/map", req.RequestURI)
		assert.Equal(t, pamTID, req.Header.Get("X-Request-Id"))

		defer req.Body.Close()
		reqBody, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, videoInvalid.BinaryContent, reqBody)

		w.WriteHeader(http.StatusBadRequest)
	}))

	log := logger.NewUPPLogger("test", "PANIC")
	validationResponse := videoInvalid.Validate(testServer.URL+"/map", tid, "", "", log)
	assert.False(t, validationResponse.IsMarkedDeleted, "Video should fail external validation.")
}

func TestIsDeleted(t *testing.T) {
	var videoNoDates = Video{
		ID:      "e28b12f7-9796-3331-b030-05082f0b8157",
		Deleted: true,
	}

	log := logger.NewUPPLogger("test", "PANIC")
	validationResponse := videoNoDates.Validate("", "", "", "", log)
	assert.True(t, validationResponse.IsMarkedDeleted, "Video should be evaluated as deleted.")
}
