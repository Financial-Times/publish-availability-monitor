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

func TestGenericContent_Validate(t *testing.T) {
	tests := map[string]struct {
		Content                        GenericContent
		ExternalValidationResponseCode int
		Expected                       ValidationResponse
	}{
		"valid generic content": {
			Content: GenericContent{
				UUID:    "077f5ac2-0491-420e-a5d0-982e0f86204b",
				Type:    "application/vnd.ft-upp-article-internal",
				Deleted: false,
				BinaryContent: []byte(`{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"title": "A title",
					"type": "Article",
					"byline": "A byline",
					"publishedDate": "2014-12-23T20:45:54.000Z",
					"firstPublishedDate": "2014-12-22T20:45:54.000Z",
					"bodyXML": "<body>Lorem ipsum</body>",
					"editorialDesk": "some string editorial desk identifier",
					"description": "Some descriptive explanation for this content",
					"mainImage": "0000aa3c-0056-506b-2b73-ed90e21b3e64",
					"someUnknownProperty" : " is totally fine, we don't validate for unknown fields/properties",
					"deleted": false
				  }`),
			},
			ExternalValidationResponseCode: http.StatusOK,
			Expected:                       ValidationResponse{IsValid: true, IsMarkedDeleted: false},
		},
		"valid deleted generic content": {
			Content: GenericContent{
				UUID:    "077f5ac2-0491-420e-a5d0-982e0f86204b",
				Type:    "application/vnd.ft-upp-article-internal",
				Deleted: true,
				BinaryContent: []byte(`{
					"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
					"title": "A title",
					"type": "Article",
					"deleted": true
				  }`),
			},
			ExternalValidationResponseCode: http.StatusOK,
			Expected:                       ValidationResponse{IsValid: true, IsMarkedDeleted: true},
		},
		"generic content with missing uuid is invalid": {
			Content: GenericContent{
				UUID: "",
			},
			ExternalValidationResponseCode: http.StatusOK,
			Expected:                       ValidationResponse{IsValid: false, IsMarkedDeleted: false},
		},
		"generic content with invalid uuid is invalid": {
			Content: GenericContent{
				UUID: "this-string-is-not-uuid",
			},
			Expected: ValidationResponse{IsValid: false, IsMarkedDeleted: false},
		},
		"generic content with failed external validatotion is invalid": {
			Content: GenericContent{
				UUID:          "077f5ac2-0491-420e-a5d0-982e0f86204b",
				Type:          "application/vnd.ft-upp-article-internal",
				BinaryContent: []byte(`invalid payload`),
			},
			ExternalValidationResponseCode: http.StatusUnsupportedMediaType,
			Expected:                       ValidationResponse{IsValid: false, IsMarkedDeleted: false},
		},
	}

	log := logger.NewUPPLogger("test", "PANIC")

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			txID := "tid_1234"
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "/validate", req.RequestURI)
				assert.Equal(t, httpcaller.ConstructPamTID(txID), req.Header.Get("X-Request-Id"))
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, test.Content.Type+"+json", req.Header.Get("Content-Type"))

				reqBody, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				defer req.Body.Close()

				assert.Equal(t, test.Content.BinaryContent, reqBody)
				w.WriteHeader(test.ExternalValidationResponseCode)
			}))

			validationResponse := test.Content.Validate(testServer.URL+"/validate", txID, "", "", log)
			assert.Equal(t, test.Content.isMarkedDeleted(), validationResponse.IsMarkedDeleted)
			assert.Equal(t, test.Expected, validationResponse)
		})
	}
}
