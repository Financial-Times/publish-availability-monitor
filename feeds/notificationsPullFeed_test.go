package feeds

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/checks"
	uuidgen "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type mockResponse struct {
	response *http.Response
	query    *url.Values
}

func buildResponse(statusCode int, content string, expectedQuery *url.Values) *mockResponse {
	return &mockResponse{
		&http.Response{
			StatusCode: statusCode,
			Body:       nopCloser{bytes.NewBuffer([]byte(content))},
		},
		expectedQuery,
	}
}

// mock httpCaller implementation
type testHTTPCaller struct {
	t             *testing.T
	authUser      string
	authPass      string
	apiKey        string
	txIdPrefix    string
	mockResponses []*mockResponse
	current       int
}

// returns the mock responses of testHTTPCaller in order
func (t *testHTTPCaller) DoCall(config checks.Config) (*http.Response, error) {
	if t.authUser != config.Username || t.authPass != config.Password {
		return buildResponse(401, `{message: "Not authenticated"}`, nil).response, nil
	}

	if t.apiKey != config.ApiKey {
		return buildResponse(401, `{"message":"No api key"}`, nil).response, nil
	}

	if t.txIdPrefix != "" {
		assert.True(t.t, strings.HasPrefix(config.TxId, t.txIdPrefix), "transaction id should start with "+t.txIdPrefix)
		timestamp := config.TxId[len(t.txIdPrefix):]
		_, err := time.Parse(time.RFC3339, timestamp)
		assert.Nil(t.t, err, "transaction id suffix did not parse as a timestamp")
	}

	response := t.mockResponses[t.current]
	if response.query != nil {
		requestUrl, _ := url.Parse(config.Url)
		assert.Equal(t.t, *response.query, requestUrl.Query())
	}

	t.current = (t.current + 1) % len(t.mockResponses)
	return response.response, nil
}

// builds testHTTPCaller with the given mocked responses in the provided order
func mockHTTPCaller(t *testing.T, txIdPrefix string, responses ...*mockResponse) checks.HttpCaller {
	return &testHTTPCaller{t: t, txIdPrefix: txIdPrefix, mockResponses: responses}
}

// builds testHTTPCaller with the given mocked responses in the provided order
func mockAuthenticatedHTTPCaller(t *testing.T, txIdPrefix string, username string, password string, apiKey string, responses ...*mockResponse) checks.HttpCaller {
	return &testHTTPCaller{t: t, txIdPrefix: txIdPrefix, authUser: username, authPass: password, apiKey: apiKey, mockResponses: responses}
}

// this is necessary to be able to build an http.Response
// the body has to be a ReadCloser
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func mockNotificationFor(uuid string, publishRef string, lastModified time.Time) string {
	return fmt.Sprintf(`{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/%v",
						"apiUrl": "http://api.ft.com/content/%v",
						"publishReference": "%v",
						"lastModified": "%v"
					}`, uuid, uuid, publishRef, lastModified.Format(time.RFC3339))
}

func mockNotificationsResponseFor(requestQueryString string, notifications string, nextLinkQueryString string) string {
	return fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?%v",
			"notifications": [
					%v
			],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?%v",
						"rel": "next"
					}
			]
		}`, requestQueryString, notifications, nextLinkQueryString)
}

func TestNotificationsArePolled(t *testing.T) {
	uuid := uuidgen.NewV4().String()
	publishRef := "tid_0123wxyz"
	lastModified := time.Now()
	notifications := mockNotificationsResponseFor("2016-10-28T15:00:00.000Z",
		mockNotificationFor(uuid, publishRef, lastModified),
		"2016-10-28T16:00:00.000Z")

	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_pull_", buildResponse(200, notifications, nil))

	baseUrl, _ := url.Parse("http://www.example.org?type=all")
	f := NewNotificationsFeed("notifications", *baseUrl, 10, 1, "", "", "")

	f.(*NotificationsPullFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()

	time.Sleep(time.Duration(1200) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
	assert.Equal(t, publishRef, response[0].PublishReference, "publish ref")
}

func TestMultipleNotificationsAreMapped(t *testing.T) {
	uuids := []string{uuidgen.NewV4().String(), uuidgen.NewV4().String()}
	publishRefs := []string{"tid_1", "tid_2"}
	lastModified := time.Now()

	notifications := mockNotificationFor(uuids[0], publishRefs[0], lastModified) + "," + mockNotificationFor(uuids[1], publishRefs[1], lastModified)

	response := mockNotificationsResponseFor("2016-10-28T15:00:00.000Z",
		notifications,
		"2016-10-28T16:00:00.000Z")

	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_pull_", buildResponse(200, response, nil))

	baseUrl, _ := url.Parse("http://www.example.org?type=all")
	f := NewNotificationsFeed("notifications", *baseUrl, 10, 1, "", "", "")

	f.(*NotificationsPullFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()

	time.Sleep(time.Duration(1200) * time.Millisecond)

	for i, _ := range uuids {
		actual := f.NotificationsFor(uuids[i])
		assert.Len(t, actual, 1, "notifications for item")
		assert.Equal(t, "http://www.ft.com/thing/"+uuids[i], actual[0].ID, "ID")
		assert.Equal(t, publishRefs[i], actual[0].PublishReference, "publish ref")
	}
}

func TestNotificationsForReturnsEmptyIfNotFound(t *testing.T) {
	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications", *baseUrl, 10, 1, "", "", "")

	response := f.NotificationsFor(uuidgen.NewV4().String())
	assert.Len(t, response, 0, "notifications for item")
}

func TestNotificationsForReturnsAllMatches(t *testing.T) {
	uuid := uuidgen.NewV4().String()
	publishRef1 := "tid_0123wxyz"
	lastModified1 := time.Now().Add(time.Duration(-1) * time.Second)
	notifications1 := mockNotificationsResponseFor("2016-10-28T15:00:00.000Z",
		mockNotificationFor(uuid, publishRef1, lastModified1),
		"2016-10-28T15:10:00.000Z")

	publishRef2 := "tid_0123abcd"
	lastModified2 := time.Now()
	notifications2 := mockNotificationsResponseFor("2016-10-28T15:10:00.000Z",
		mockNotificationFor(uuid, publishRef2, lastModified2),
		"2016-10-28T15:20:00.000Z")

	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_pull_", buildResponse(200, notifications1, nil), buildResponse(200, notifications2, nil))

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications", *baseUrl, 10, 1, "", "", "")
	f.(*NotificationsPullFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()
	time.Sleep(time.Duration(2200) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 2, "notifications for item")
	assert.Equal(t, publishRef1, response[0].PublishReference, "first publish ref")
	assert.Equal(t, publishRef2, response[1].PublishReference, "second publish ref")
}

func TestNotificationsPollingContinuesAfterErrorResponse(t *testing.T) {
	uuid := uuidgen.NewV4().String()
	publishRef := "tid_0123wxyz"
	lastModified := time.Now()
	notifications := mockNotificationsResponseFor("2016-10-28T15:00:00.000Z",
		mockNotificationFor(uuid, publishRef, lastModified),
		"2016-10-28T16:00:00.000Z")

	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_pull_", buildResponse(500, "", nil), buildResponse(200, notifications, nil))

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications", *baseUrl, 10, 1, "", "", "")
	f.(*NotificationsPullFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()
	time.Sleep(time.Duration(2200) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
	assert.Equal(t, publishRef, response[0].PublishReference, "publish ref")
}

func TestNotificationsArePurged(t *testing.T) {
	uuid := uuidgen.NewV4().String()
	publishRef := "tid_0123wxyz"
	lastModified := time.Now().Add(time.Duration(-2) * time.Second)
	notifications := mockNotificationsResponseFor("2016-10-28T15:00:00.000Z",
		mockNotificationFor(uuid, publishRef, lastModified),
		"2016-10-28T16:00:00.000Z")

	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_pull_", buildResponse(200, notifications, nil))

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications", *baseUrl, 1, 1, "", "", "")
	f.(*NotificationsPullFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()

	time.Sleep(time.Duration(1200) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
	assert.Equal(t, publishRef, response[0].PublishReference, "publish ref")

	time.Sleep(time.Duration(1) * time.Second)
	response = f.NotificationsFor(uuid)
	assert.Len(t, response, 0, "notifications for item")
}

func TestNotificationsPollingFollowsOpaqueLink(t *testing.T) {
	uuid1 := uuidgen.NewV4().String()
	publishRef1 := "tid_0123wxyz"
	lastModified1 := time.Now().Add(time.Duration(-1) * time.Second)
	bootstrapQuery := url.Values{"since": []string{"any"}}
	nextPageQuery := url.Values{"page": []string{"12345"}}

	notifications1 := mockNotificationsResponseFor(bootstrapQuery.Encode(),
		mockNotificationFor(uuid1, publishRef1, lastModified1),
		nextPageQuery.Encode())

	uuid2 := uuidgen.NewV4().String()
	publishRef2 := "tid_0123abcd"
	lastModified2 := time.Now()
	notifications2 := mockNotificationsResponseFor(nextPageQuery.Encode(),
		mockNotificationFor(uuid2, publishRef2, lastModified2),
		"page=xxx")

	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_pull_", buildResponse(200, notifications1, nil), buildResponse(200, notifications2, &nextPageQuery))

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications", *baseUrl, 10, 1, "", "", "")
	f.(*NotificationsPullFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()
	time.Sleep(time.Duration(2200) * time.Millisecond)

	response1 := f.NotificationsFor(uuid1)
	assert.Len(t, response1, 1, "notifications for "+uuid1)
	assert.Equal(t, publishRef1, response1[0].PublishReference, "publish ref for "+uuid1)

	response2 := f.NotificationsFor(uuid2)
	assert.Len(t, response2, 1, "notifications for "+uuid2)
	assert.Equal(t, publishRef2, response2[0].PublishReference, "publish ref for "+uuid2)
}
