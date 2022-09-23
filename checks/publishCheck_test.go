package checks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	"github.com/stretchr/testify/assert"
)

func TestIsCurrentOperationFinished_ContentCheck_InvalidContent(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1234"`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.False(t, finished, "Expected error.")
}

func TestIsCurrentOperationFinished_ContentCheck_Finished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, finished, "operation should have finished successfully")
}

func TestIsCurrentOperationFinished_ContentCheck_WithAuthentication(t *testing.T) {
	currentTid := "tid_5678"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	username := "jdoe"
	password := "frodo"
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockAuthenticatedHTTPCaller(t, "tid_pam_5678", username, password, response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		username,
		password,
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, finished, "operation should have finished successfully")
}

func TestIsCurrentOperationFinished_ContentCheck_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235"}`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.False(t, finished, "Expected failure.")
}

func TestIsCurrentOperationFinished_ContentCheck_MarkedDeleted_Finished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	response := buildResponse(404, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, finished, "operation should have finished successfully.")
}

func TestIsCurrentOperationFinished_ContentCheck_MarkedDeleted_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.False(t, finished, "operation should not have finished")
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsAfterCurrentPublishDate_IgnoreCheckTrue(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(DateLayout, "2016-01-08T14:22:06.271Z")
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-01-08T14:22:07.391Z" }`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	_, ignoreCheck := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, ignoreCheck, "check should be ignored")
}

// fails for dateLayout="2006-01-02T15:04:05.000Z"
func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsBeforeCurrentPublishDateSpecifiedWith2Decimals_IgnoreCheckFalse(t *testing.T) {
	currentTid := "tid_1234"

	publishDate, err := time.Parse(DateLayout, "2016-02-01T14:30:21.55Z")
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-02-01T14:30:21.549Z" }`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	_, ignoreCheck := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.False(t, ignoreCheck, "check should not be ignored")
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsBeforeCurrentPublishDate_IgnoreCheckFalse(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(DateLayout, "2016-01-08T14:22:06.271Z")
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-01-08T14:22:05.391Z" }`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	_, ignoreCheck := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.False(t, ignoreCheck, "check should not be ignored")
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsBeforeCurrentPublishDate_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(DateLayout, "2016-01-08T14:22:06.271Z")
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-01-08T14:22:05.391Z" }`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.False(t, finished, "operation should not have finished")
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateEqualsCurrentPublishDate_Finished(t *testing.T) {
	currentTid := "tid_1234"
	publishDateAsString := "2016-01-08T14:22:06.271Z"
	publishDate, err := time.Parse(DateLayout, publishDateAsString)
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s", "lastModified" : "%s" }`, currentTid, publishDateAsString)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, finished, "operation should have finished successfully")
}

// fallback to publish reference check if last modified date is not valid
func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsNullCurrentTIDAndPubReferenceMatch_Finished(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(DateLayout, "2016-01-08T14:22:06.271Z")
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s", "lastModified" : null }`, currentTid)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, finished, "operation should have finished successfully")
}

// fallback to publish reference check if last modified date is not valid
func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsEmptyStringCurrentTIDAndPubReferenceMatch_Finished(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(DateLayout, "2016-01-08T14:22:06.271Z")
	assert.Nil(t, err, "Failure in setting up test data")

	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s", "lastModified" : "" }`, currentTid)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(
		pm,
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	))
	assert.True(t, finished, "operation should have finished successfully")
}

type publishMetricBuilder interface {
	withUUID(string) publishMetricBuilder
	withEndpoint(string) publishMetricBuilder
	withPlatform(string) publishMetricBuilder
	withTID(string) publishMetricBuilder
	withMarkedDeleted(bool) publishMetricBuilder
	withPublishDate(time.Time) publishMetricBuilder
	build() metrics.PublishMetric
}

// PublishMetricBuilder implementation
type pmBuilder struct {
	UUID          string
	endpoint      url.URL
	platform      string
	tid           string
	markedDeleted bool
	publishDate   time.Time
}

func (b *pmBuilder) withUUID(uuid string) publishMetricBuilder {
	b.UUID = uuid
	return b
}

func (b *pmBuilder) withEndpoint(endpoint string) publishMetricBuilder {
	e, _ := url.Parse(endpoint)
	b.endpoint = *e
	return b
}

func (b *pmBuilder) withPlatform(platform string) publishMetricBuilder {
	b.platform = platform
	return b
}

func (b *pmBuilder) withTID(tid string) publishMetricBuilder {
	b.tid = tid
	return b
}

func (b *pmBuilder) withMarkedDeleted(markedDeleted bool) publishMetricBuilder {
	b.markedDeleted = markedDeleted
	return b
}

func (b *pmBuilder) withPublishDate(publishDate time.Time) publishMetricBuilder {
	b.publishDate = publishDate
	return b
}

func (b *pmBuilder) build() metrics.PublishMetric {
	return metrics.PublishMetric{
		UUID:            b.UUID,
		Endpoint:        b.endpoint,
		Platform:        b.platform,
		TID:             b.tid,
		IsMarkedDeleted: b.markedDeleted,
		PublishDate:     b.publishDate,
	}
}

func newPublishMetricBuilder() publishMetricBuilder {
	return &pmBuilder{}
}

func buildResponse(statusCode int, content string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBuffer([]byte(content))),
	}
}

// mock httpCaller implementation
type testHTTPCaller struct {
	t             *testing.T
	authUser      string
	authPass      string
	tid           string
	mockResponses []*http.Response
	current       int
}

// returns the mock responses of testHTTPCaller in order
func (t *testHTTPCaller) DoCall(config httpcaller.Config) (*http.Response, error) {
	if t.authUser != config.Username || t.authPass != config.Password {
		return buildResponse(401, `{message: "Not authenticated"}`), nil
	}

	if t.tid != "" {
		assert.Equal(t.t, t.tid, config.TID, "transaction id")
	}

	response := t.mockResponses[t.current]
	t.current = (t.current + 1) % len(t.mockResponses)
	return response, nil
}

// builds testHTTPCaller with the given mocked responses in the provided order
func mockHTTPCaller(t *testing.T, txID string, responses ...*http.Response) httpcaller.Caller {
	return &testHTTPCaller{t: t, tid: txID, mockResponses: responses}
}

// builds testHTTPCaller with the given mocked responses in the provided order
func mockAuthenticatedHTTPCaller(t *testing.T, txID string, username string, password string, responses ...*http.Response) httpcaller.Caller {
	return &testHTTPCaller{t: t, tid: txID, authUser: username, authPass: password, mockResponses: responses}
}
