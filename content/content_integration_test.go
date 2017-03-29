package content

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
)

func TestThat_ContentPayloadFromKafkaMsg_MatchesTheContentPayloadSentToValidationServices(t *testing.T) {
	var testCases = []struct {
		name string
		msg  consumer.Message
	}{
		{
			"Methode List",
			consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				},
				Body: string(loadBytesForFile(t, "methode_list.json")),
			},
		},
		{
			"Methode Empty List",
			consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				},
				Body: string(loadBytesForFile(t, "methode_empty_list.json")),
			},
		},
		{
			"Methode Article",
			consumer.Message{
				Headers: map[string]string{
					"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				},
				Body: string(loadBytesForFile(t, "methode_article.json")),
			},
		},
	}

	for _, tc := range testCases {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			checkSameContentBody(t, strings.TrimSpace(tc.msg.Body), strings.TrimSpace(string(body)), "[%s] testcase failure", tc.name)
		}))
		defer ts.Close()
		content, err := UnmarshalContent(tc.msg)
		if err != nil {
			t.Errorf("Expected success, but error occured [%v]", err)
			return
		}
		//don't care about the actual result
		content.Validate(ts.URL, "tid_foobar", "", "")
	}
}

func checkSameContentBody(t *testing.T, expectedBody string, actualBody string, msgAndArgs ...interface{}) {
	var expectedContent EomFile
	err := json.Unmarshal([]byte(expectedBody), &expectedContent)
	if err != nil {
		t.Fatal(err)
	}

	var actualContent EomFile
	err = json.Unmarshal([]byte(actualBody), &actualContent)
	assert.NoError(t, err, "It should not return an unmarshalling error")

	assert.Equal(t, expectedContent.UUID, actualContent.UUID, "They should have same UUID")
	assert.Equal(t, expectedContent.Attributes, actualContent.Attributes, "They should have same Attributes")
	assert.Equal(t, expectedContent.ContentType, actualContent.ContentType, "They should have same ContentType")
	assert.Equal(t, expectedContent.SystemAttributes, actualContent.SystemAttributes, "They should have same SystemAttributes")
	assert.Equal(t, expectedContent.Value, actualContent.Value, "They should have same Value")
	assert.Equal(t, expectedContent.UsageTickets, actualContent.UsageTickets, "They should have same UsageTickets")
	assert.Equal(t, expectedContent.WorkflowStatus, actualContent.WorkflowStatus, "They should have same WorkflowStatus")
	assert.Equal(t, expectedContent.LinkedObjects, actualContent.LinkedObjects, "They should have same LinkedObjects")
}
