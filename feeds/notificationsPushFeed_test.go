package feeds

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockPushNotificationsStream struct {
	notifications []string
	index         int
}

func (resp *mockPushNotificationsStream) Read(p []byte) (n int, err error) {
	var data []byte
	if resp.index >= len(resp.notifications) {
		data = []byte("data: []\n")
	} else {
		data = []byte("data: [" + resp.notifications[resp.index] + "]\n")
		resp.index++
		log.Infof("data: %v", string(data))
	}
	actual := len(data)
	for i := 0; i < actual; i++ {
		p[i] = data[i]
	}

	return actual, nil
}

func (resp *mockPushNotificationsStream) Close() error {
	return nil
}

func buildPushResponse(statusCode int, notifications []string) (*mockResponse, *mockPushNotificationsStream) {
	stream := &mockPushNotificationsStream{notifications, 0}
	return &mockResponse{
		&http.Response{
			StatusCode: statusCode,
			Body:       stream,
		}, nil}, stream
}

func TestPushNotificationsAreConsumed(t *testing.T) {
	uuid := "1cb14245-5185-4ed5-9188-4d2a86085599"
	publishRef := "tid_0123wxyz"
	lastModified := time.Now()
	notifications := mockNotificationFor(uuid, publishRef, lastModified)
	notifications = strings.Replace(notifications, "\n", "", -1)

	httpResponse, _ := buildPushResponse(200, []string{notifications})
	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_push_", httpResponse)

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications-push", *baseUrl, 10, 1, "", "", "")
	f.(*NotificationsPushFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()

	time.Sleep(time.Duration(100) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
	assert.Equal(t, publishRef, response[0].PublishReference, "publish ref")
}

func TestPushNotificationsForReturnsEmptyIfNotFound(t *testing.T) {
	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications-push", *baseUrl, 10, 1, "", "", "")

	response := f.NotificationsFor("1cb14245-5185-4ed5-9188-4d2a86085599")
	assert.Len(t, response, 0, "notifications for item")
}

func TestPushNotificationsForReturnsAllMatches(t *testing.T) {
	uuid := "1cb14245-5185-4ed5-9188-4d2a86085599"
	publishRef1 := "tid_0123wxyz"
	lastModified1 := time.Now().Add(time.Duration(-1) * time.Second)
	notification1 := mockNotificationFor(uuid, publishRef1, lastModified1)

	publishRef2 := "tid_0123abcd"
	lastModified2 := time.Now()
	notification2 := mockNotificationFor(uuid, publishRef2, lastModified2)

	httpResponses, _ := buildPushResponse(200, []string{
		strings.Replace(notification1, "\n", "", -1),
		strings.Replace(notification2, "\n", "", -1),
	})
	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_push_", httpResponses)

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications-push", *baseUrl, 10, 1, "", "", "")
	f.(*NotificationsPushFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()
	time.Sleep(time.Duration(2200) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 2, "notifications for item")
	assert.Equal(t, publishRef1, response[0].PublishReference, "first publish ref")
	assert.Equal(t, publishRef2, response[1].PublishReference, "second publish ref")
}

func TestPushNotificationsPollingContinuesAfterErrorResponse(t *testing.T) {
	uuid := "1cb14245-5185-4ed5-9188-4d2a86085599"
	publishRef := "tid_0123wxyz"
	lastModified := time.Now()
	notification := mockNotificationFor(uuid, publishRef, lastModified)

	httpResponse, _ := buildPushResponse(200, []string{strings.Replace(notification, "\n", "", -1)})
	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_push_", buildResponse(500, "", nil), httpResponse)

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications-push", *baseUrl, 10, 1, "", "", "")
	f.(*NotificationsPushFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()
	time.Sleep(time.Duration(550) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
	assert.Equal(t, publishRef, response[0].PublishReference, "publish ref")
}

func TestPushNotificationsArePurged(t *testing.T) {
	uuid := "1cb14245-5185-4ed5-9188-4d2a86085599"
	publishRef := "tid_0123wxyz"
	lastModified := time.Now().Add(time.Duration(-2) * time.Second)
	notifications := mockNotificationFor(uuid, publishRef, lastModified)
	notifications = strings.Replace(notifications, "\n", "", -1)

	httpResponse, _ := buildPushResponse(200, []string{notifications})
	httpCaller := mockHTTPCaller(t, "tid_pam_notifications_push_", httpResponse)

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications-push", *baseUrl, 1, 1, "", "", "")
	f.(*NotificationsPushFeed).SetHttpCaller(httpCaller)
	f.Start()
	defer f.Stop()

	time.Sleep(time.Duration(500) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
	assert.Equal(t, publishRef, response[0].PublishReference, "publish ref")

	time.Sleep(time.Duration(2) * time.Second)
	response = f.NotificationsFor(uuid)
	assert.Len(t, response, 0, "notifications for item")
}

func TestPushNotificationsSendsAuthentication(t *testing.T) {
	uuid := "1cb14245-5185-4ed5-9188-4d2a86085599"
	publishRef := "tid_0123wxyz"
	lastModified := time.Now()
	notifications := mockNotificationFor(uuid, publishRef, lastModified)
	notifications = strings.Replace(notifications, "\n", "", -1)

	httpResponse, _ := buildPushResponse(200, []string{notifications})

	baseUrl, _ := url.Parse("http://www.example.org")
	f := NewNotificationsFeed("notifications-push", *baseUrl, 10, 1, "someUser", "somePwd", "someApiKey")
	httpCaller := mockAuthenticatedHTTPCaller(t, "tid_pam_notifications_push_", "someUser", "somePwd", "someApiKey", httpResponse)
	f.(*NotificationsPushFeed).SetHttpCaller(httpCaller)

	f.Start()
	defer f.Stop()

	time.Sleep(time.Duration(500) * time.Millisecond)

	response := f.NotificationsFor(uuid)
	assert.Len(t, response, 1, "notifications for item")
}
