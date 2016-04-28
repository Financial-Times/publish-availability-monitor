package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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
	infoLogger.Printf("Running check for %s\n", loggingContextForCheck(pc.Metric.config.Alias, pc.Metric.UUID, pc.Metric.tid))
	check := endpointSpecificChecks[pc.Metric.config.Alias]
	if check == nil {
		warnLogger.Printf("No check for %s", loggingContextForCheck(pc.Metric.config.Alias, pc.Metric.UUID, pc.Metric.tid))
		return false, false
	}

	return check.isCurrentOperationFinished(pc.Metric)
}

func (c ContentCheck) isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool) {
	url := pm.endpoint.String() + pm.UUID
	resp, err := c.httpCaller.doCall(url)
	if err != nil {
		warnLogger.Printf("Error calling URL: [%v] for %s : [%v]", url, loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), err.Error())
		return false, false
	}
	defer cleanupResp(resp)

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.isMarkedDeleted {
		infoLogger.Printf("Content Marked deleted. Checking %s, status code [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), resp.StatusCode)
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
		warnLogger.Printf("Checking %s. Cannot read response: [%s]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), err.Error())
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		warnLogger.Printf("Checking %s. Cannot unmarshal JSON response: [%s]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), err.Error())
		return false, false
	}

	// look for rapid-fire publishes
	lastModifiedDate, ok := parseLastModifiedDate(jsonResp)
	if ok {
		if (*lastModifiedDate).After(pm.publishDate) {
			infoLogger.Printf("Checking %s. Last modified date [%v] is after publish date [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), lastModifiedDate, pm.publishDate)
			return false, true
		}
		if (*lastModifiedDate).Equal(pm.publishDate) {
			infoLogger.Printf("Checking %s. Last modified date [%v] is equal to publish date [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), lastModifiedDate, pm.publishDate)
			return true, false
		}
		infoLogger.Printf("Checking %s. Last modified date [%v] is before publish date [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), lastModifiedDate, pm.publishDate)
		return false, false
	}
	warnLogger.Printf("The field 'lastModified' is not valid: [%v]. Skip checking rapid-fire publishes for %s.", jsonResp["lastModified"], loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid))

	// fallback check
	infoLogger.Printf("Checking %s. Fallback checking publishReference [%v] from response.", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), jsonResp["publishReference"])
	return jsonResp["publishReference"] == pm.tid, false
}

func parseLastModifiedDate(jsonContent map[string]interface{}) (*time.Time, bool) {
	lastModifiedDateAsString, ok := jsonContent["lastModified"].(string)
	if ok && lastModifiedDateAsString != "" {
		lastModifiedDate, err := time.Parse(dateLayout, lastModifiedDateAsString)
		return &lastModifiedDate, err == nil
	}
	return nil, false
}

// ignoreCheck is always false
func (s S3Check) isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool) {
	url := pm.endpoint.String() + pm.UUID
	resp, err := s.httpCaller.doCall(url)
	if err != nil {
		warnLogger.Printf("Checking %s. Error calling URL: [%v] : [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), url, err.Error())
		return false, false
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		return false, false
	}

	// we have to check if the body is null because of an issue where the image is
	// uploaded to S3, but body is empty - in this case, we get 200 back but empty body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		warnLogger.Printf("Checking %s. Cannot read response: [%s]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), err.Error())
		return false, false
	}

	if len(data) == 0 {
		warnLogger.Printf("Checking %s. Image body is empty!", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid))
		return false, false
	}
	return true, false
}

// ignore unused field (e.g. requestUrl)
type notificationsContent struct {
	Notifications []notification
	Links         []link
}

// ignore unused fields (e.g. type, apiUrl)
type notification struct {
	PublishReference string
	LastModified     string
	ID               string
}

// ignore unused field (e.g. rel)
type link struct {
	Href string
}

func (n NotificationsCheck) isCurrentOperationFinished(pm PublishMetric) (operationFinished, ignoreCheck bool) {
	if n.shouldSkipCheck(pm) {
		return false, true
	}
	notificationsURL := buildNotificationsURL(pm)
	var err error
	for {
		result := n.checkBatchOfNotifications(notificationsURL, pm)
		if result.operationFinished || result.nextNotificationsURL == "" {
			return result.operationFinished, result.ignoreCheck
		}
		//replace nextNotificationsURL host, as by default it's the API gateway host
		notificationsURL, err = adjustNextNotificationsURL(notificationsURL, result.nextNotificationsURL)
		if err != nil {
			return false, false
		}
	}
}

func (n NotificationsCheck) shouldSkipCheck(pm PublishMetric) bool {
	if !pm.isMarkedDeleted {
		return false
	}
	url := pm.endpoint.String() + "/" + pm.UUID
	resp, err := n.httpCaller.doCall(url)
	if err != nil {
		warnLogger.Printf("Checking %s. Error calling URL: [%v] : [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), url, err.Error())
		return false
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		return false
	}

	var notifications []notification
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		return false
	}
	//ignore check if there are no previous notifications for this UUID
	if len(notifications) == 0 {
		return true
	}

	return false
}

// Bundles the result of a single check of batch of notifications
// Note: the next notifications URL has the host set to the API gateway host, as returned by the notifications service.
type notificationCheckResult struct {
	// the status of the check
	operationFinished bool
	// true, if there is a more recent notification found for the current content
	ignoreCheck bool
	// the URL for the next batch of notifications to be checked (if it is applicable), otherwise empty string ""
	nextNotificationsURL string
}

// Check the notification content with the provided publishReference from the batch of notifications from the provided URL.
// Returns the status of the check and the
func (n NotificationsCheck) checkBatchOfNotifications(notificationsURL string, pm PublishMetric) notificationCheckResult {
	// return this check result where is appropriate (e.g. in case of errors): operation not finished, ignore checks false, empty next notifications URL
	var defaultResult = notificationCheckResult{operationFinished: false, ignoreCheck: false, nextNotificationsURL: ""}

	resp, err := n.httpCaller.doCall(notificationsURL)
	if err != nil {
		warnLogger.Printf("Checking %s. Error calling URL: [%v] : [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), notificationsURL, err.Error())
		return defaultResult
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		warnLogger.Printf("Checking %s. Status NOT OK: [%d]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), resp.StatusCode)
		return defaultResult
	}

	var notifications notificationsContent
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		warnLogger.Printf("Checking %s. Cannot decode json response: [%s]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), err.Error())
		return defaultResult
	}
	for _, n := range notifications.Notifications {
		if !strings.Contains(n.ID, pm.UUID) {
			continue
		}

		lastModifiedDate, err := time.Parse(dateLayout, n.LastModified)
		if err != nil {
			warnLogger.Printf("The field 'lastModified' is not valid: [%v]. Skip checking rapid-fire publishes for %s.", n.LastModified, loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid))
			//fallback check
			infoLogger.Printf("Checking %s. Fallback checking publishReference [%v] from response.", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), n.PublishReference)
			return notificationCheckResult{operationFinished: pm.tid == n.PublishReference, ignoreCheck: false, nextNotificationsURL: ""}
		}
		if lastModifiedDate.After(pm.publishDate) {
			infoLogger.Printf("Checking %s. Last modified date [%v] is after publish date [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), lastModifiedDate, pm.publishDate)
			return notificationCheckResult{operationFinished: false, ignoreCheck: true, nextNotificationsURL: ""}
		}
		if lastModifiedDate.Equal(pm.publishDate) {
			infoLogger.Printf("Checking %s. Last modified date [%v] is equal to publish date [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), lastModifiedDate, pm.publishDate)
			return notificationCheckResult{operationFinished: true, ignoreCheck: false, nextNotificationsURL: ""}
		}
		infoLogger.Printf("Checking %s. Last modified date [%v] is before publish date [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.tid), lastModifiedDate, pm.publishDate)
		return defaultResult
	}

	if len(notifications.Notifications) > 0 {
		return notificationCheckResult{operationFinished: false, ignoreCheck: false, nextNotificationsURL: notifications.Links[0].Href}
	}
	return defaultResult
}

func buildNotificationsURL(pm PublishMetric) string {
	base := pm.endpoint.String()
	queryParam := url.Values{}
	//e.g. 2015-07-23T00:00:00.000Z
	since := pm.publishDate.Format(dateLayout)
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

func cleanupResp(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}
	err = resp.Body.Close()
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}
}
