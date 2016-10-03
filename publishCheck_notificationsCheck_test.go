package main

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

func TestCheckBatchOfNotifications_ResponseBatchOfNotificationsIsEmpty_NotFinished(t *testing.T) {
	testResponse := `{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
			"notifications": [],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); result.operationFinished {
		t.Error("Expected failure")
	}
}

func TestCheckBatchOfNotifications_ResponseDoesNotContainUUID_NotFinished(t *testing.T) {
	testResponse := `{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085599",
						"apiUrl": "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085599",
						"publishReference": "tid_0123wxyz",
						"lastModified": "2015-12-08T16:16:47.391Z"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID("1cb14245-5185-4ed5-9188-4d2a86085598").withTID("tid_0123wxyz").build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); result.operationFinished {
		t.Error("Expected failure")
	}
}

// fallback to tid check in case of lastModified field parsing error
func TestCheckBatchOfNotifications_ResponseDoesContainUUIDLastModifiedFieldIsUnparseableTIDsDiffer_NotFinished(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085599"
	testResponse := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/%s",
						"apiUrl": "http://api.ft.com/content/%s",
						"publishReference": "tid_0123wxyZ",
						"lastModified": "2015-12-08T1616:47.391Z"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`, currentUUID, currentUUID)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(currentUUID).withTID("tid_0123wxyz").build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); result.operationFinished {
		t.Error("Expected failure")
	}
}

// fallback to tid check in case of lastModified field parsing error
func TestCheckBatchOfNotifications_ResponseDoesContainUUIDLastModifiedFieldIsUnparseableTIDsMatch_Finished(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085599"
	currentTID := "tid_0123wxyZ"
	testResponse := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/%s",
						"apiUrl": "http://api.ft.com/content/%s",
						"publishReference": "%s",
						"lastModified": "2015-12-08T1616:47.391Z"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`, currentUUID, currentUUID, currentTID)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(currentUUID).withTID(currentTID).build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); !result.operationFinished {
		t.Error("Expected success")
	}
}

func TestCheckBatchOfNotifications_ResponseDoesContainUUIDLastModifiedFieldIsAfterCurrentPublishedDate_IgnoreCheck(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085599"
	currentPublishedDate, err := time.Parse(dateLayout, "2015-12-08T16:16:47.391Z")
	if err != nil {
		t.Error("Error in setting up test data")
		return
	}
	testResponse := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/%s",
						"apiUrl": "http://api.ft.com/content/%s",
						"publishReference": "tid_foo",
						"lastModified": "2015-12-08T16:16:48.391Z"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`, currentUUID, currentUUID)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(currentUUID).withPublishDate(currentPublishedDate).build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); !result.ignoreCheck {
		t.Error("Expected ignoreCheck to be [true].")
	}
}

func TestCheckBatchOfNotifications_ResponseDoesContainUUIDLastModifiedFieldEqualsCurrentPublishedDate_Finished(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085599"
	currentPublishedDate, err := time.Parse(dateLayout, "2015-12-08T16:16:47.391Z")
	if err != nil {
		t.Error("Error in setting up test data")
		return
	}
	testResponse := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/%s",
						"apiUrl": "http://api.ft.com/content/%s",
						"lastModified": "2015-12-08T16:16:47.391Z",
						"publishReference":"tid_foo"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`, currentUUID, currentUUID)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(currentUUID).withPublishDate(currentPublishedDate).build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); !result.operationFinished {
		t.Error("Expected success.")
	}
}

func TestCheckBatchOfNotifications_ResponseDoesContainUUIDLastModifiedFieldBeforeCurrentPublishedDate_NotFinished(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085599"
	currentPublishedDate, err := time.Parse(dateLayout, "2015-12-08T16:16:47.391Z")
	if err != nil {
		t.Error("Error in setting up test data")
		return
	}
	testResponse := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/%s",
						"apiUrl": "http://api.ft.com/content/%s",
						"lastModified": "2015-12-08T16:16:46.391Z",
						"publishReference": "tid_foo"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`, currentUUID, currentUUID)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(currentUUID).withPublishDate(currentPublishedDate).build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); result.operationFinished {
		t.Error("Expected failure.")
	}
}

func TestCheckBatchOfNotifications_ResponseIsNotEmptyDoesNotContainUUID_NextNotificationsURLIsSet(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085598"
	nextNotificationsURL := "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z"
	testResponse := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085599",
						"apiUrl": "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085599",
						"lastModified": "2015-12-08T16:16:46.391Z"
					}
				],
			"links": [
					{
						"href": "%s",
						"rel": "next"
					}
			]
		}`, nextNotificationsURL)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withUUID(currentUUID).build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); result.nextNotificationsURL != nextNotificationsURL {
		t.Error("Expected success.")
	}
}

func TestCheckBatchOfNotifications_ResponseBatchOfNotificationsIsEmpty_NextNotificationsURLIsEmptyString(t *testing.T) {
	testResponse := fmt.Sprint(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`)
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().build(), "", "", 0, 0, nil)
	if result := notificationsCheck.checkBatchOfNotifications("dummy-url", pc); result.nextNotificationsURL != "" {
		t.Error("Expected empty string.")
	}
}

func TestIsCurrentOperationFinished_FirstBatchOfNotificationsContainsUUIDAndTID_Finished(t *testing.T) {
	testTID := "tid_0123wxyz"
	testResponse := fmt.Sprintf(
		`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085599",
						"apiUrl": "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085599",
						"publishReference": "%s"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`, testTID)

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse)),
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withTID(testTID).build(), "", "", 0, 0, nil)
	if finished, _ := notificationsCheck.isCurrentOperationFinished(pc); !finished {
		t.Error("Expected success")
	}
}

func TestIsCurrentOperationFinished_FirstBatchOfNotificationsDoesNotContainUUIDSecondBatchIsEmpty_NotFinished(t *testing.T) {
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085598"
	testResponse1 := `{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085599",
						"apiUrl": "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085599",
						"publishReference": "tid_0123wxyz"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
 						"rel": "next"
					}
			]
		}`
	testResponse2 := `{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
			"notifications": [],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`
	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse1), buildResponse(200, testResponse2)),
	}
	pc := NewPublishCheck(newPublishMetricBuilder().withTID("tid_0123wxyZ").withUUID(currentUUID).build(), "", "", 0, 0, nil)
	if finished, _ := notificationsCheck.isCurrentOperationFinished(pc); finished {
		t.Error("Expected failure")
	}
}

func TestIsCurrentOperationFinished_FirstBatchOfNotificationsDoesNotContainUUIDButSecondDoes_Finished(t *testing.T) {
	currentTID := "tid_0123wxyZ"
	currentUUID := "1cb14245-5185-4ed5-9188-4d2a86085599"
	testResponse1 := `{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T00:00:00.000Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085598",
						"apiUrl": "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085598",
						"publishReference": "tid_0123wxyz"
					}
				],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
						"rel": "next"
					}
			]
		}`
	testResponse2 := fmt.Sprintf(`{
			"requestUrl": "http://api.ft.com/content/notifications?since=2015-11-09T14:09:08.705Z",
			"notifications": [
					{
						"type": "http://www.ft.com/thing/ThingChangeType/UPDATE",
						"id": "http://www.ft.com/thing/1cb14245-5185-4ed5-9188-4d2a86085599",
						"apiUrl": "http://api.ft.com/content/1cb14245-5185-4ed5-9188-4d2a86085599",
						"publishReference": "%s"
					}
			],
			"links": [
					{
						"href": "http://api.ft.com/content/notifications?since=2015-11-09T14:10:08.500Z",
						"rel": "next"
					}
			]
		}`, currentTID)

	notificationsCheck := &NotificationsCheck{
		mockHTTPCaller(buildResponse(200, testResponse1), buildResponse(200, testResponse2)),
	}

	pc := NewPublishCheck(newPublishMetricBuilder().withTID(currentTID).withUUID(currentUUID).build(), "", "", 0, 0, nil)
	if finished, _ := notificationsCheck.isCurrentOperationFinished(pc); !finished {
		t.Error("Expected success")
	}
}

func TestNotificationsBuildURL_Success(test *testing.T) {
	publishDate, err := time.Parse(dateLayout, "2015-10-21T14:22:06.271Z")
	if err != nil {
		test.Errorf("Test data error: [%v]", err)
	}
	pm := newPublishMetricBuilder().withEndpoint("http://notifications-endpoint:8080/content/notifications").withPublishDate(publishDate).build()

	expected := "http://notifications-endpoint:8080/content/notifications?since=2015-10-21T14:22:06.271Z"

	actual, err := url.QueryUnescape(buildNotificationsURL(pm))
	if err != nil {
		test.Errorf("Expected success. Found error: [%v]", err)
	}
	if actual != expected {
		test.Errorf("Expected success.\nActual: [%s]\nExpected: [%s]", actual, expected)
	}
}

func TestAdjustNextNotificationsURL_CurrentHostAndPortDiffers_Success(t *testing.T) {
	current := "http://ftapp14927-lvpr-uk-int:8080/content/notifications?since=2015-12-15T00:00:00.000Z"
	next := "http://int.api.ft.com/content/notifications?since=2015-12-15T11:53:17.508Z"

	expected := "http://ftapp14927-lvpr-uk-int:8080/content/notifications?since=2015-12-15T11:53:17.508Z"

	actual, err := adjustNextNotificationsURL(current, next)
	if err != nil {
		t.Errorf("Expected success. Found error: [%v]", err)
	}

	if actual != expected {
		t.Error("Expected success")
	}
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
		mockHTTPCaller(buildResponse(200, `[{"id": "foobar", "lastModified" : "foobaz", "publishReference" : "unitTestRef" }]`)),
	}
	if notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected failure")
	}
}

func TestShouldSkipCheck_ContentIsMarkedAsDeletedPreviousNotificationsDoesNotExist_CheckSkipped(t *testing.T) {
	pm := newPublishMetricBuilder().withMarkedDeleted(true).withEndpoint("http://notifications-endpoint:8080/content/notifications").build()
	pc := NewPublishCheck(pm, "", "", 0, 0, nil)
	notificationsCheck := NotificationsCheck{
		mockHTTPCaller(buildResponse(200, `[]`)),
	}
	if !notificationsCheck.shouldSkipCheck(pc) {
		t.Errorf("Expected success")
	}
}

func TestCheckNotificationItems_ShouldNotFinishOpIfNotFound(t *testing.T) {
	notifications := notificationsContent{
		Notifications: []notification{
			notification{
				ID:               "0dda9446-4367-11e6-b22f-79eb4891c97d",
				LastModified:     "2016-07-07T07:07:07.000Z",
				PublishReference: "tid_testtest1",
			},
			notification{
				ID:               "0dda9446-4367-11e6-b22f-79eb4891c97d",
				LastModified:     "2016-07-07T07:07:08.000Z",
				PublishReference: "tid_testtest2",
			},
		},
		Links: []link{
			link{
				Href: "",
			},
		},
	}
	pubDate, err := time.Parse(dateLayout, "2016-07-07T07:07:09.000Z")
	if err != nil {
		t.Errorf(err.Error())
	}
	var pm PublishMetric
	pm = PublishMetric{
		UUID:        "0dda9446-4367-11e6-b22f-79eb4891c97d",
		publishDate: pubDate,
		platform:    "test",
		//publishInterval:
		tid:             "tid_testtest99",
		isMarkedDeleted: false,
	}
	pc := NewPublishCheck(pm, "", "", 0, 0, nil)
	var defaultResult = notificationCheckResult{operationFinished: false, ignoreCheck: false, nextNotificationsURL: ""}
	actual := checkNotificationItems(notifications, pc, defaultResult)

	if actual != defaultResult {
		t.Errorf("Should not signal the finish of operation or ignore the check. Actual: %v", actual)
	}
}
