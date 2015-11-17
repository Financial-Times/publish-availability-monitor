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
	s3Check := &S3Check{}
	if !s3Check.isCurrentOperationFinished(PublishMetric{}, buildResponse(200, "imagebytes")) {
		t.Errorf("Expected: true. Actual: false")
	}
}

func TestIsCurrentOperationFinished_S3Check_Empty(t *testing.T) {
	s3Check := &S3Check{}
	if s3Check.isCurrentOperationFinished(PublishMetric{}, buildResponse(200, "")) {
		t.Errorf("Expected: false. Actual: true")
	}
}

func TestIsCurrentOperationFinished_S3Check_NotFinished(t *testing.T) {
	s3Check := &S3Check{}
	if s3Check.isCurrentOperationFinished(PublishMetric{}, buildResponse(404, "")) {
		t.Errorf("Expected: false. Actual: True")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_InvalidContent(t *testing.T) {
	contentCheck := &ContentCheck{}
	testResponse := `{ "uuid" : "1234-1234"`

	if contentCheck.isCurrentOperationFinished(PublishMetric{}, buildResponse(200, testResponse)) {
		t.Errorf("Expected error.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_Finished(t *testing.T) {
	contentCheck := &ContentCheck{}

	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)

	if !contentCheck.isCurrentOperationFinished(newPublishMetricBuilder().withTID(currentTid).build(), buildResponse(200, testResponse)) {
		t.Error("Expected success.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_NotFinished(t *testing.T) {
	contentCheck := &ContentCheck{}

	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235"}`
	fmt.Println(testResponse)
	if contentCheck.isCurrentOperationFinished(newPublishMetricBuilder().withTID(currentTid).build(), buildResponse(200, testResponse)) {
		t.Error("Expected failure.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_MarkedDeleted_Finished(t *testing.T) {
	contentCheck := &ContentCheck{}

	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)

	if !contentCheck.isCurrentOperationFinished(newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build(), buildResponse(404, testResponse)) {
		t.Error("Expected success.")
	}
}

func TestIsCurrentOperationFinished_ContentCheck_MarkedDeleted_NotFinished(t *testing.T) {
	contentCheck := &ContentCheck{}

	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)

	if contentCheck.isCurrentOperationFinished(newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build(), buildResponse(200, testResponse)) {
		t.Error("Expected failure.")
	}
}

func TestIsCurrentOperaitonFinished_NotificationsCheck_ResponseContainsUUID_Finished(t *testing.T) {
	notificationsCheck := &NotificationsCheck{}

	testUUID := "1234-4321"
	testResponse := fmt.Sprintf(
		`{ 
			requestUrl: "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			notifications: [
					{
						type: "http://www.ft.com/thing/ThingChangeType/UPDATE",
						id: "http://www.ft.com/thing/%s",
						apiUrl: "http://api.ft.com/content/sameuuidhere"
					},
				],
			links: [
					{
						href: "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						rel: "next"
					}
			]
		}`, testUUID)

	if !notificationsCheck.isCurrentOperationFinished(newPublishMetricBuilder().withUUID(testUUID).build(), buildResponse(200, testResponse)) {
		t.Error("Expected success")
	}
}

func TestIsCurrentOperaitonFinished_NotificationsCheck_ResponseDoesNotContainUUID_NotFinished(t *testing.T) {
	notificationsCheck := &NotificationsCheck{}

	testUUID := "1234-4321"
	testResponse := fmt.Sprint(
		`{ 
			requestUrl: "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			notifications: [
					{
						type: "http://www.ft.com/thing/ThingChangeType/UPDATE",
						id: "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085599",
						apiUrl: "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085599"
					},
				],
			links: [
					{
						href: "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						rel: "next"
					}
			]
		}`)

	if notificationsCheck.isCurrentOperationFinished(newPublishMetricBuilder().withUUID(testUUID).build(), buildResponse(200, testResponse)) {
		t.Error("Expected failure")
	}
}
func TestBuildURL_NotificationsCheck(test *testing.T) {
	nc := &NotificationsCheck{}

	publishDate, err := time.Parse(time.RFC3339Nano, "2015-10-21T14:22:06.270Z")
	if err != nil {
		test.Errorf("Error in test data: [%v]", err)
	}

	pm := newPublishMetricBuilder().withEndpoint("http://notifications-endpoint:8080/content/notifications").withPublishDate(publishDate).build()

	builtURL, err := url.Parse(nc.buildURL(pm))
	if err != nil {
		test.Errorf("Cannot parse built URL: [%s].", err.Error())
	}

	queryParams := builtURL.Query()
	since := queryParams.Get("since")
	if since == "" {
		test.Errorf("Missing 'since' query parameter.")
	}

	t, err := time.Parse(time.RFC3339Nano, since)
	if err != nil {
		test.Errorf("Cannot parse param value: [%s]. Error: [%s]", since, err.Error())
	}

	if !t.Equal(publishDate) {
		test.Errorf("Expected timestamp: [%v]. Actual: [%v].", publishDate, t)
	}
}

func TestBuildURL_S3Check_And_ContentCheck(test *testing.T) {
	tests := []struct {
		check    EndpointSpecificCheck
		pm       PublishMetric
		expected string
	}{
		{
			&S3Check{},
			newPublishMetricBuilder().withUUID("1234-1234").withEndpoint("https://s3-image-check/test/").build(),
			"https://s3-image-check/test/1234-1234",
		},
		{
			&ContentCheck{},
			newPublishMetricBuilder().withUUID("4321-1234").withEndpoint("http://content-check/content-read/").build(),
			"http://content-check/content-read/4321-1234",
		},
	}

	for _, t := range tests {
		actual := t.check.buildURL(t.pm)
		if actual != t.expected {
			test.Errorf("Expected: [%s]. Actual: [%s].", t.expected, actual)
		}
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

// this is necessary to be able to build an http.Response
// the body has to be a ReadCloser
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
