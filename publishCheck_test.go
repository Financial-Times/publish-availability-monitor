package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestIsCurrentOperationFinished_S3Check_Finished(t *testing.T) {
	s3Check := &S3Check{
		mockHTTPCaller(buildResponse(200, "imagebytes")),
	}
	if finished, _ := s3Check.isCurrentOperationFinished(NewPublishCheck(PublishMetric{}, "", "", 0, 0)); !finished {
		t.Errorf("Expected: true. Actual: false")
	}
}

func TestIsCurrentOperationFinished_S3Check_DoesNotSendAuthentication(t *testing.T) {
	currentTid := "tid_1234"
	s3Check := &S3Check{
		mockHTTPCaller(buildResponse(200, "imagebytes")),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	pc := NewPublishCheck(pm, "jdoe", "frodo", 0, 0)
	if finished, _ := s3Check.isCurrentOperationFinished(pc); !finished {
		t.Error("Expected success.")
	}
}

func TestIsCurrentOperationFinished_S3Check_Empty(t *testing.T) {
	s3Check := &S3Check{
		mockHTTPCaller(buildResponse(200, "")),
	}
	if finished, _ := s3Check.isCurrentOperationFinished(NewPublishCheck(PublishMetric{}, "", "", 0, 0)); finished {
		t.Errorf("Expected: false. Actual: true")
	}
}

func TestIsCurrentOperationFinished_S3Check_NotFinished(t *testing.T) {
	s3Check := &S3Check{
		mockHTTPCaller(buildResponse(404, "")),
	}
	if finished, _ := s3Check.isCurrentOperationFinished(NewPublishCheck(PublishMetric{}, "", "", 0, 0)); finished {
		t.Errorf("Expected: false. Actual: True")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_InvalidContent(t *testing.T) {
	testResponse := `{ "uuid" : "1234-1234"`
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(PublishMetric{}, "", "", 0, 0)); finished {
		t.Errorf("Expected error.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_Finished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); !finished {
		t.Error("Expected success.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_WithAuthentication(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	username := "jdoe"
	password := "frodo"
	contentCheck := &ContentCheck{
		mockAuthenticatedHTTPCaller(username, password, buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, username, password, 0, 0)); !finished {
		t.Error("Expected success.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235"}`
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); finished {
		t.Error("Expected failure.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_MarkedDeleted_Finished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(404, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); !finished {
		t.Error("Expected success.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_MarkedDeleted_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); finished {
		t.Error("Expected failure.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsAfterCurrentPublishDate_IgnoreCheckTrue(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(dateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprint(`{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-01-08T14:22:07.391Z" }`)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if _, ignoreCheck := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); !ignoreCheck {
		t.Error("Expected ignoreCheck to be true")
	}
}

//fails for dateLayout="2006-01-02T15:04:05.000Z"
func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsBeforeCurrentPublishDateSpecifiedWith2Decimals_IgnoreCheckFalse(t *testing.T) {
	currentTid := "tid_1234"

	publishDate, err := time.Parse(dateLayout, "2016-02-01T14:30:21.55Z")
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprint(`{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-02-01T14:30:21.549Z" }`)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if _, ignoreCheck := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); ignoreCheck {
		t.Error("Expected ignoreCheck to be false")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsBeforeCurrentPublishDate_IgnoreCheckFalse(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(dateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprint(`{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-01-08T14:22:05.391Z" }`)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if _, ignoreCheck := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); ignoreCheck {
		t.Error("Expected ignoreCheck to be false.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsBeforeCurrentPublishDate_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(dateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprint(`{ "uuid" : "1234-1234", "publishReference" : "tid_1235", "lastModified" : "2016-01-08T14:22:05.391Z" }`)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); finished {
		t.Error("Expected failure.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateEqualsCurrentPublishDate_Finished(t *testing.T) {
	currentTid := "tid_1234"
	publishDateAsString := "2016-01-08T14:22:06.271Z"
	publishDate, err := time.Parse(dateLayout, publishDateAsString)
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s", "lastModified" : "%s" }`, currentTid, publishDateAsString)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); !finished {
		t.Error("Expected success.")
	}
}

// fallback to publish reference check if last modified date is not valid
func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsNullCurrentTIDAndPubReferenceMatch_Finished(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(dateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s", "lastModified" : null }`, currentTid)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); !finished {
		t.Error("Expected success.")
	}
}

// fallback to publish reference check if last modified date is not valid
func TestIsCurrentOperationFinished_ContentCheck_LastModifiedDateIsEmptyStringCurrentTIDAndPubReferenceMatch_Finished(t *testing.T) {
	currentTid := "tid_1234"
	publishDate, err := time.Parse(dateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		t.Error("Failure in setting up test data")
		return
	}
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s", "lastModified" : "" }`, currentTid)
	contentCheck := &ContentCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pm := newPublishMetricBuilder().withTID(currentTid).withPublishDate(publishDate).build()
	if finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0)); !finished {
		t.Error("Expected success.")
	}
}

type publishMetricBuilder interface {
	withUUID(string) publishMetricBuilder
	withEndpoint(string) publishMetricBuilder
	withTID(string) publishMetricBuilder
	withMarkedDeleted(bool) publishMetricBuilder
	withPublishDate(time.Time) publishMetricBuilder
	build() PublishMetric
}

//PublishMetricBuilder implementation
type pmBuilder struct {
	UUID          string
	endpoint      url.URL
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

func (b *pmBuilder) build() PublishMetric {
	return PublishMetric{
		UUID:            b.UUID,
		endpoint:        b.endpoint,
		tid:             b.tid,
		isMarkedDeleted: b.markedDeleted,
		publishDate:     b.publishDate,
	}
}

func newPublishMetricBuilder() publishMetricBuilder {
	return &pmBuilder{}
}

func buildResponse(statusCode int, content string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       nopCloser{bytes.NewBuffer([]byte(content))},
	}
}

// mock httpCaller implementation
type testHTTPCaller struct {
	authUser      string
	authPass      string
	mockResponses []*http.Response
	current       int
}

// returns the mock responses of testHTTPCaller in order
func (t *testHTTPCaller) doCall(url string, username string, password string) (*http.Response, error) {
	if t.authUser != username || t.authPass != password {
		return buildResponse(401, `{message: "Not authenticated"}`), nil
	}

	response := t.mockResponses[t.current]
	t.current = (t.current + 1) % len(t.mockResponses)
	return response, nil
}

// builds testHTTPCaller with the given mocked responses in the provided order
func mockHTTPCaller(responses ...*http.Response) httpCaller {
	return &testHTTPCaller{mockResponses: responses}
}

// builds testHTTPCaller with the given mocked responses in the provided order
func mockAuthenticatedHTTPCaller(username string, password string, responses ...*http.Response) httpCaller {
	return &testHTTPCaller{authUser: username, authPass: password, mockResponses: responses}
}

// this is necessary to be able to build an http.Response
// the body has to be a ReadCloser
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
