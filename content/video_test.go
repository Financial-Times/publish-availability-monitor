package content

import (
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsVideoValid_Valid(t *testing.T) {
	var videoValid = Video{
		UUID:          "e28b12f7-9796-3331-b030-05082f0b8157",
		Id:            "4966650664001",
		Name:          "the-dark-knight.mp4",
		UpdatedAt:     "2016-06-01T21:40:19.120Z",
		PublishedAt:   "2016-06-01T21:40:19.120Z",
		BinaryContent: []byte("valid-json"),
	}

	txId := "tid_1234"
	pamTxId := checks.ConstructPamTxId(txId)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/map", req.RequestURI)
		assert.Equal(t, pamTxId, req.Header.Get("X-Request-Id"))

		defer req.Body.Close()
		reqBody, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, videoValid.BinaryContent, reqBody)
	}))

	validationResponse := videoValid.Validate(testServer.URL+"/map", txId, "", "")
	if !validationResponse.IsValid {
		t.Error("Video should be valid.")
	}
}

func TestIsVideoValid_NoId(t *testing.T) {
	var videoNoId = Video{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		Name:        "the-dark-knight.mp4",
		UpdatedAt:   "2016-06-01T21:40:19.120Z",
		PublishedAt: "2016-06-01T21:40:19.120Z",
	}

	validationResponse := videoNoId.Validate("", "", "", "")
	if validationResponse.IsValid {
		t.Error("Video should be invalid as it has no Id.")
	}
}

func TestIsVideoValid_NoUUID(t *testing.T) {
	var videoNoUUID = Video{
		Id:          "4966650664001",
		Name:        "the-dark-knight.mp4",
		UpdatedAt:   "2016-06-01T21:40:19.120Z",
		PublishedAt: "2016-06-01T21:40:19.120Z",
	}

	validationResponse := videoNoUUID.Validate("", "", "", "")
	if validationResponse.IsValid {
		t.Error("Video should be invalid as it has no uuid.")
	}
}

func TestIsVideoValid_failedExternalValidation(t *testing.T) {
	var videoInvalid = Video{
		UUID:          "e28b12f7-9796-3331-b030-05082f0b8157",
		Id:            "4966650664001",
		Name:          "the-dark-knight.mp4",
		UpdatedAt:     "2016-06-01T21:40:19.120Z",
		PublishedAt:   "2016-06-01T21:40:19.120Z",
		BinaryContent: []byte("invalid-json"),
	}

	txId := "tid_1234"
	pamTxId := checks.ConstructPamTxId(txId)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/map", req.RequestURI)
		assert.Equal(t, pamTxId, req.Header.Get("X-Request-Id"))

		defer req.Body.Close()
		reqBody, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, videoInvalid.BinaryContent, reqBody)

		w.WriteHeader(http.StatusBadRequest)
	}))

	validationResponse := videoInvalid.Validate(testServer.URL+"/map", txId, "", "")
	if validationResponse.IsMarkedDeleted {
		t.Error("Video should fail external validation.")
	}
}

func TestIsDeleted_NoDates(t *testing.T) {
	var videoNoDates = Video{
		UUID: "e28b12f7-9796-3331-b030-05082f0b8157",
		Id:   "4966650664001",
		Name: "the-dark-knight.mp4",
	}

	validationResponse := videoNoDates.Validate("", "", "", "")
	if !validationResponse.IsMarkedDeleted {
		t.Error("Video should be evaluated as deleted as it has no dates in it.")
	}
}

func TestIsDeleted_OneDateOnly(t *testing.T) {
	var videoOneDateOnly = Video{
		UUID:      "e28b12f7-9796-3331-b030-05082f0b8157",
		Id:        "4966650664001",
		Name:      "the-dark-knight.mp4",
		UpdatedAt: "2016-06-01T21:40:19.120Z",
	}

	validationResponse := videoOneDateOnly.Validate("", "", "", "")
	if validationResponse.IsMarkedDeleted {
		t.Error("Video should be evaluated as published as it has one date in it.")
	}
}
