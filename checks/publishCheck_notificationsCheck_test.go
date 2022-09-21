package checks

import (
	"testing"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	testEnv  = "testEnv"
	feedName = "testFeed"
)

type testFeed struct {
	feedName      string
	feedType      string
	uuid          string
	notifications []*feeds.Notification
}

func (f testFeed) Start() {}
func (f testFeed) Stop()  {}
func (f testFeed) FeedName() string {
	return f.feedName
}
func (f testFeed) FeedType() string {
	return f.feedType
}
func (f testFeed) FeedURL() string {
	return f.feedName
}
func (f testFeed) SetCredentials(string, string) {}
func (f testFeed) NotificationsFor(uuid string) []*feeds.Notification {
	return f.notifications
}

func mockFeed(name string, uuid string, notifications []*feeds.Notification) testFeed {
	return testFeed{name, feeds.NotificationsPull, uuid, notifications}
}

func TestFeedContainsMatchingNotification(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID := "tid_0123wxyz"
	testLastModified := "2016-10-28T14:00:00.000Z"

	n := feeds.Notification{ID: testUUID, PublishReference: testTxID, LastModified: testLastModified}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUUID, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	finished, _ := notificationsCheck.isCurrentOperationFinished(pc)
	assert.True(t, finished, "Operation should be considered finished")
}

func TestFeedMissingNotification(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID := "tid_0123wxyz"

	f := mockFeed(feedName, uuid.NewString(), []*feeds.Notification{})
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	finished, _ := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
}

func TestFeedContainsEarlierNotification(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID1 := "tid_0123abcd"
	testLastModified1 := "2016-10-28T13:59:00.000Z"
	testTxID2 := "tid_0123wxyz"
	testLastModified2, _ := time.Parse(DateLayout, "2016-10-28T14:00:00.000Z")

	n := feeds.Notification{ID: testUUID, PublishReference: testTxID1, LastModified: testLastModified1}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUUID, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID2).withPublishDate(testLastModified2).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be skipped")
}

func TestFeedContainsLaterNotification(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID1 := "tid_0123abcd"
	testLastModified1 := "2016-10-28T14:00:00.000Z"
	testTxID2 := "tid_0123wxyz"
	testLastModified2, _ := time.Parse(DateLayout, "2016-10-28T13:59:00.000Z")

	n := feeds.Notification{ID: testUUID, PublishReference: testTxID1, LastModified: testLastModified1}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUUID, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID2).withPublishDate(testLastModified2).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	_, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.True(t, ignore, "Operation should be skipped")
}

func TestFeedContainsUnparseableNotification(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID1 := "tid_0123abcd"
	testLastModified1 := "foo-bar-baz"
	testTxID2 := "tid_0123wxyz"
	testLastModified2, _ := time.Parse(DateLayout, "2016-10-28T13:59:00.000Z")

	n := feeds.Notification{ID: testUUID, PublishReference: testTxID1, LastModified: testLastModified1}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUUID, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID2).withPublishDate(testLastModified2).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be skipped")
}

func TestMissingFeed(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID := "tid_0123wxyz"
	testLastModified := "2016-10-28T14:00:00.000Z"

	n := feeds.Notification{ID: testUUID, PublishReference: testTxID, LastModified: testLastModified}
	notifications := []*feeds.Notification{&n}
	f := mockFeed("foo", testUUID, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be ignored")
}

func TestMissingEnvironment(t *testing.T) {
	testUUID := uuid.NewString()
	testTxID := "tid_0123wxyz"
	testLastModified := "2016-10-28T14:00:00.000Z"

	n := feeds.Notification{ID: testUUID, PublishReference: testTxID, LastModified: testLastModified}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUUID, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds["foo"] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(
		newPublishMetricBuilder().withUUID(testUUID).withPlatform(testEnv).withTID(testTxID).build(),
		"",
		"",
		0,
		0,
		nil,
		nil,
		log,
	)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be ignored")
}

func TestShouldSkipCheck_ContentIsNotMarkedAsDeleted_CheckNotSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(false).build()
	notificationsCheck := NotificationsCheck{}
	log := logger.NewUPPLogger("test", "PANIC")

	pc := NewPublishCheck(pm, "", "", 0, 0, nil, nil, log)

	if notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected failure")
	}
}

func TestShouldSkipCheck_ContentIsMarkedAsDeletedPreviousNotificationsExist_CheckNotSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(true).withEndpoint("http://notifications-endpoint:8080/content/notifications").build()
	log := logger.NewUPPLogger("test", "PANIC")
	pc := NewPublishCheck(pm, "", "", 0, 0, nil, nil, log)
	notificationsCheck := NotificationsCheck{
		mockHTTPCaller(t, "", buildResponse(200, `[{"id": "foobar", "lastModified" : "foobaz", "publishReference" : "unitTestRef" }]`)), nil, feedName,
	}
	if notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected failure")
	}
}

func TestShouldSkipCheck_ContentIsMarkedAsDeletedPreviousNotificationsDoesNotExist_CheckSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(true).withEndpoint("http://notifications-endpoint:8080/content/notifications").build()
	log := logger.NewUPPLogger("test", "PANIC")
	pc := NewPublishCheck(pm, "", "", 0, 0, nil, nil, log)
	notificationsCheck := NotificationsCheck{
		mockHTTPCaller(t, "", buildResponse(200, `[]`)), nil, feedName,
	}
	if !notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected success")
	}
}
