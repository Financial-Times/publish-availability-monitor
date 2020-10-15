package main

import (
	"testing"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/feeds"
	"github.com/satori/go.uuid"
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
	testUuid := uuid.NewV4().String()
	testTxID := "tid_0123wxyz"
	testLastModified := "2016-10-28T14:00:00.000Z"

	n := feeds.Notification{ID: testUuid, PublishReference: testTxID, LastModified: testLastModified}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUuid, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID).build(), "", "", 0, 0, nil)
	finished, _ := notificationsCheck.isCurrentOperationFinished(pc)
	assert.True(t, finished, "Operation should be considered finished")
}

func TestFeedMissingNotification(t *testing.T) {
	testUuid := uuid.NewV4().String()
	testTxID := "tid_0123wxyz"

	f := mockFeed(feedName, uuid.NewV4().String(), []*feeds.Notification{})
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID).build(), "", "", 0, 0, nil)
	finished, _ := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
}

func TestFeedContainsEarlierNotification(t *testing.T) {
	testUuid := uuid.NewV4().String()
	testTxID1 := "tid_0123abcd"
	testLastModified1 := "2016-10-28T13:59:00.000Z"
	testTxID2 := "tid_0123wxyz"
	testLastModified2, _ := time.Parse(dateLayout, "2016-10-28T14:00:00.000Z")

	n := feeds.Notification{ID: testUuid, PublishReference: testTxID1, LastModified: testLastModified1}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUuid, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID2).withPublishDate(testLastModified2).build(), "", "", 0, 0, nil)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be skipped")
}

func TestFeedContainsLaterNotification(t *testing.T) {
	testUuid := uuid.NewV4().String()
	testTxID1 := "tid_0123abcd"
	testLastModified1 := "2016-10-28T14:00:00.000Z"
	testTxID2 := "tid_0123wxyz"
	testLastModified2, _ := time.Parse(dateLayout, "2016-10-28T13:59:00.000Z")

	n := feeds.Notification{ID: testUuid, PublishReference: testTxID1, LastModified: testLastModified1}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUuid, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID2).withPublishDate(testLastModified2).build(), "", "", 0, 0, nil)
	_, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.True(t, ignore, "Operation should be skipped")
}

func TestFeedContainsUnparseableNotification(t *testing.T) {
	testUuid := uuid.NewV4().String()
	testTxID1 := "tid_0123abcd"
	testLastModified1 := "foo-bar-baz"
	testTxID2 := "tid_0123wxyz"
	testLastModified2, _ := time.Parse(dateLayout, "2016-10-28T13:59:00.000Z")

	n := feeds.Notification{ID: testUuid, PublishReference: testTxID1, LastModified: testLastModified1}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUuid, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID2).withPublishDate(testLastModified2).build(), "", "", 0, 0, nil)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be skipped")
}

func TestMissingFeed(t *testing.T) {
	testUuid := uuid.NewV4().String()
	testTxID := "tid_0123wxyz"
	testLastModified := "2016-10-28T14:00:00.000Z"

	n := feeds.Notification{ID: testUuid, PublishReference: testTxID, LastModified: testLastModified}
	notifications := []*feeds.Notification{&n}
	f := mockFeed("foo", testUuid, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds[testEnv] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID).build(), "", "", 0, 0, nil)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be ignored")
}

func TestMissingEnvironment(t *testing.T) {
	testUuid := uuid.NewV4().String()
	testTxID := "tid_0123wxyz"
	testLastModified := "2016-10-28T14:00:00.000Z"

	n := feeds.Notification{ID: testUuid, PublishReference: testTxID, LastModified: testLastModified}
	notifications := []*feeds.Notification{&n}
	f := mockFeed(feedName, testUuid, notifications)
	subscribedFeeds := make(map[string][]feeds.Feed)
	subscribedFeeds["foo"] = []feeds.Feed{f}

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(t, "", nil),
		subscribedFeeds,
		feedName,
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(testUuid).withPlatform(testEnv).withTID(testTxID).build(), "", "", 0, 0, nil)
	finished, ignore := notificationsCheck.isCurrentOperationFinished(pc)
	assert.False(t, finished, "Operation should not be considered finished")
	assert.False(t, ignore, "Operation should not be ignored")
}

func TestShouldSkipCheck_ContentIsNotMarkedAsDeleted_CheckNotSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(false).build()
	notificationsCheck := NotificationsCheck{}
	pc := NewPublishCheck(pm, "", "", 0, 0, nil)

	if notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected failure")
	}
}

func TestShouldSkipCheck_ContentIsMarkedAsDeletedPreviousNotificationsExist_CheckNotSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(true).withEndpoint("http://notifications-endpoint:8080/content/notifications").build()
	pc := NewPublishCheck(pm, "", "", 0, 0, nil)
	notificationsCheck := NotificationsCheck{
		mockHTTPCaller(t, "", buildResponse(200, `[{"id": "foobar", "lastModified" : "foobaz", "publishReference" : "unitTestRef" }]`)), nil, feedName,
	}
	if notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected failure")
	}
}

func TestShouldSkipCheck_ContentIsMarkedAsDeletedPreviousNotificationsDoesNotExist_CheckSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(true).withEndpoint("http://notifications-endpoint:8080/content/notifications").build()
	pc := NewPublishCheck(pm, "", "", 0, 0, nil)
	notificationsCheck := NotificationsCheck{
		mockHTTPCaller(t, "", buildResponse(200, `[]`)), nil, feedName,
	}
	if !notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected success")
	}
}
