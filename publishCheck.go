package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// PublishCheck performs an availability  check on a piece of content, at a
// given endpoint, and returns whether the check was successful or not.
// Holds all the information necessary to check content availability
// at an endpoint, as well as store and send the results of the check.
type PublishCheck struct {
	Metric        PublishMetric
	Threshold     int
	CheckInterval int
	ResultSink    chan PublishMetric
}

// EndpointSpecificCheck is the interface which determines the state of the operation we are currently checking.
type EndpointSpecificCheck interface {
	// Returns the state of the operation and whether this check should be ignored
	isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool)
}

// ContentCheck implements the EndpointSpecificCheck interface to check operation
// status for the content endpoint.
type ContentCheck struct {
	httpCaller httpCaller
}

// S3Check implements the EndpointSpecificCheck interface to check operation
// status for the S3 endpoint.
type S3Check struct {
	httpCaller httpCaller
}

// NotificationsCheck implements the EndpointSpecificCheck interface to build the endpoint URL and
// to check the operation is present in the notification feed
type NotificationsCheck struct {
	httpCaller httpCaller
}

// httpCaller abstracts http calls
type httpCaller interface {
	doCall(url string) (*http.Response, error)
}

// Default implementation of httpCaller
type defaultHTTPCaller struct{}

// Performs http GET calls using the default http client
func (c defaultHTTPCaller) doCall(url string) (resp *http.Response, err error) {
	return http.Get(url)
}

// NewPublishCheck returns a PublishCheck ready to perform a check for pm.UUID, at the
// pm.endpoint.
func NewPublishCheck(pm PublishMetric, t int, ci int, rs chan PublishMetric) *PublishCheck {
	return &PublishCheck{pm, t, ci, rs}
}

var endpointSpecificChecks map[string]EndpointSpecificCheck

func init() {
	hC := defaultHTTPCaller{}

	//key is the endpoint alias from the config
	endpointSpecificChecks = map[string]EndpointSpecificCheck{
		"content":         ContentCheck{hC},
		"S3":              S3Check{hC},
		"enrichedContent": ContentCheck{hC},
		"lists":           ContentCheck{hC},
		"notifications":   NotificationsCheck{hC},
	}
}

// DoCheck performs an availability check on a piece of content at a certain
// endpoint, applying endpoint-specific processing.
// Returns true if the content is available at the endpoint, false otherwise.
func (pc PublishCheck) DoCheck() (checkSuccessful, ignoreCheck bool) {
	infoLogger.Printf("Running [%s] check for UUID [%v]\n", pc.Metric.config.Alias, pc.Metric.UUID)
	check := endpointSpecificChecks[pc.Metric.config.Alias]
	if check == nil {
		warnLogger.Printf("No check for endpoint %s.", pc.Metric.config.Alias)
		return false, false
	}

	return check.isCurrentOperationFinished(pc.Metric)
}

func (c ContentCheck) isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool) {
	url := pm.endpoint.String() + pm.UUID
	resp, err := c.httpCaller.doCall(url)
	if err != nil {
		warnLogger.Printf("Error calling URL: [%v] : [%v]", url, err.Error())
		return false, false
	}
	defer resp.Body.Close()

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.isMarkedDeleted {
		infoLogger.Printf("[%v]Marked deleted, status code [%v]", pm.UUID, resp.StatusCode)
		return resp.StatusCode == 404, false
	}

	// if not marked deleted, operation isn't finished until status is 200
	if resp.StatusCode != 200 {
		return false, false
	}

	// if status is 200, we check the publishReference
	// this way we can handle updates
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		warnLogger.Printf("Cannot read response: [%s]", err.Error())
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		warnLogger.Printf("Cannot unmarshal JSON response: [%s]", err.Error())
		return false, false
	}

	// look for rapid-fire publishes
	lastModifiedDateAsString, ok := jsonResp["lastModified"].(string)
	if ok && lastModifiedDateAsString != "" {
		lastModifiedDate, err := time.Parse(dateLayout, lastModifiedDateAsString)
		if err != nil {
			errorLogger.Printf("Cannot parse publish date [%v] from message [%v], error: [%v]",
				jsonResp["lastModified"], pm.tid, err.Error())
			return false, false
		}
		if lastModifiedDate.After(pm.publishDate) {
			return false, true
		}
	} else {
		warnLogger.Printf("Skip checking rapid-fire publishes for UUID [%s]. The field 'lastModified' is not valid: [%v].", pm.UUID, jsonResp["lastModified"])
	}

	return jsonResp["publishReference"] == pm.tid, false
}

// ignoreCheck is always false
func (s S3Check) isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool) {
	url := pm.endpoint.String() + pm.UUID
	resp, err := s.httpCaller.doCall(url)
	if err != nil {
		warnLogger.Printf("Error calling URL: [%v] : [%v]", url, err.Error())
		return false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, false
	}

	// we have to check if the body is null because of an issue where the image is
	// uploaded to S3, but body is empty - in this case, we get 200 back but empty body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		warnLogger.Printf("Cannot read response: [%s]", err.Error())
		return false, false
	}

	if len(data) == 0 {
		warnLogger.Printf("Image [%v] body is empty!", pm.UUID)
		return false, false
	}
	return true, false
}

// ignore unused field (e.g. requestUrl)
type notificationsContent struct {
	Notifications []notifications
	Links         []link
}

// ignore unused fields (e.g. type, id, apiUrl)
type notifications struct {
	PublishReference string
}

// ignore unused field (e.g. rel)
type link struct {
	Href string
}

// ignoreCheck is always false
func (n NotificationsCheck) isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool) {
	notificationsURL := buildNotificationsURL(pm)
	var err error
	for {
		finished, nextNotificationsURL := n.checkBatchOfNotifications(notificationsURL, pm.tid)
		if finished || nextNotificationsURL == "" {
			return finished, false
		}
		//replace nextNotificationsURL host, as by default it's the API gateway host
		notificationsURL, err = adjustNextNotificationsURL(notificationsURL, nextNotificationsURL)
		if err != nil {
			return false, false
		}
	}
}

// Check the notification content with the provided publishReference from the batch of notifications from the provided URL.
// Returns the status of the check and the URL for the next batch of notifications (if it is applicable).
// Note: the next notifications URL has the host set to the API gateway host, as returned by the notifications service.
func (n NotificationsCheck) checkBatchOfNotifications(notificationsURL, publishReference string) (bool, string) {
	resp, err := n.httpCaller.doCall(notificationsURL)
	if err != nil {
		warnLogger.Printf("Error calling URL: [%v] : [%v]", notificationsURL, err.Error())
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		warnLogger.Printf("/notifications endpoint status: [%d]", resp.StatusCode)
		return false, ""
	}

	var notifications notificationsContent
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		warnLogger.Printf("Cannot decode json response: [%s]", err.Error())
		return false, ""
	}
	for _, n := range notifications.Notifications {
		if n.PublishReference == publishReference {
			return true, ""
		}
	}

	if len(notifications.Notifications) > 0 {
		return false, notifications.Links[0].Href
	}
	return false, ""
}

func buildNotificationsURL(pm PublishMetric) string {
	base := pm.endpoint.String()
	queryParam := url.Values{}
	//e.g. 2015-07-23T00:00:00.000Z
	since := pm.publishDate.Format(time.RFC3339Nano)
	queryParam.Add("since", since)
	return base + "?" + queryParam.Encode()
}

// Replace next URL host with current URL's host
func adjustNextNotificationsURL(current, next string) (string, error) {
	currentNotificationsURLValue, err := url.Parse(current)
	if err != nil {
		warnLogger.Printf("Cannot parse current notifications URL: [%s].", current)
		return "", err
	}
	nextNotificationsURLValue, err := url.Parse(next)
	if err != nil {
		warnLogger.Printf("Cannot parse next notifications URL: [%s].", next)
		return "", err
	}
	nextNotificationsURLValue.Host = currentNotificationsURLValue.Host
	return nextNotificationsURLValue.String(), nil
}
