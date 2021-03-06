package checks

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	log "github.com/Sirupsen/logrus"
)

const DateLayout = time.RFC3339Nano

// PublishCheck performs an availability  check on a piece of content, at a
// given endpoint, and returns whether the check was successful or not.
// Holds all the information necessary to check content availability
// at an endpoint, as well as store and send the results of the check.
type PublishCheck struct {
	Metric                 metrics.PublishMetric
	username               string
	password               string
	Threshold              int
	CheckInterval          int
	ResultSink             chan metrics.PublishMetric
	endpointSpecificChecks map[string]EndpointSpecificCheck
}

// NewPublishCheck returns a PublishCheck ready to perform a check for pm.UUID, at the
// pm.Endpoint.
func NewPublishCheck(pm metrics.PublishMetric, username string, password string, t int, ci int, rs chan metrics.PublishMetric, endpointSpecificChecks map[string]EndpointSpecificCheck) *PublishCheck {
	return &PublishCheck{pm, username, password, t, ci, rs, endpointSpecificChecks}
}

// DoCheck performs an availability check on a piece of content at a certain
// endpoint, applying endpoint-specific processing.
// Returns true if the content is available at the endpoint, false otherwise.
func (pc PublishCheck) DoCheck() (checkSuccessful, ignoreCheck bool) {
	log.Infof("Running check for %s\n", pc)
	check := pc.endpointSpecificChecks[pc.Metric.Config.Alias]
	if check == nil {
		log.Warnf("No check for %s", pc)
		return false, false
	}

	return check.isCurrentOperationFinished(&pc)
}

func (pc PublishCheck) String() string {
	return LoggingContextForCheck(pc.Metric.Config.Alias, pc.Metric.UUID, pc.Metric.Platform, pc.Metric.TID)
}

// EndpointSpecificCheck is the interface which determines the state of the operation we are currently checking.
type EndpointSpecificCheck interface {
	// Returns the state of the operation and whether this check should be ignored
	isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool)
}

// ContentCheck implements the EndpointSpecificCheck interface to check operation
// status for the content endpoint.
type ContentCheck struct {
	httpCaller httpcaller.Caller
}

func NewContentCheck(httpCaller httpcaller.Caller) ContentCheck {
	return ContentCheck{httpCaller: httpCaller}
}

func (c ContentCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	url := pm.Endpoint.String() + pm.UUID
	resp, err := c.httpCaller.DoCall(httpcaller.Config{URL: url, Username: pc.username, Password: pc.password, TxID: httpcaller.ConstructPamTxId(pm.TID)}) //nolint:bodyclose
	if err != nil {
		log.Warnf("Error calling URL: [%v] for %s : [%v]", url, pc, err.Error())
		return false, false
	}
	defer cleanupResp(resp)

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.IsMarkedDeleted {
		log.Infof("Content Marked deleted. Checking %s, status code [%v]", pc, resp.StatusCode)
		return resp.StatusCode == 404, false
	}

	// if not marked deleted, operation isn't finished until status is 200
	if resp.StatusCode != 200 {
		if resp.StatusCode != 404 {
			log.Infof("Checking %s, status code [%v]", pc, resp.StatusCode)
		}
		return false, false
	}

	// if status is 200, we check the publishReference
	// this way we can handle updates
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Checking %s. Cannot read response: [%s]",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), err.Error())
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		log.Warnf("Checking %s. Cannot unmarshal JSON response: [%s]",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), err.Error())
		return false, false
	}

	return isSamePublishEvent(jsonResp, pc)
}

// ContentNeo4jCheck implements the EndpointSpecificCheck interface to check operation
// status for the content endpoint.
type ContentNeo4jCheck struct {
	httpCaller httpcaller.Caller
}

func NewContentNeo4jCheck(httpCaller httpcaller.Caller) ContentNeo4jCheck {
	return ContentNeo4jCheck{httpCaller: httpCaller}
}

func (c ContentNeo4jCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	url := pm.Endpoint.String() + pm.UUID

	resp, err := c.httpCaller.DoCall(httpcaller.Config{ //nolint:bodyclose
		URL:      url,
		Username: pc.username,
		Password: pc.password,
		TxID:     httpcaller.ConstructPamTxId(pm.TID)})

	if err != nil {
		log.Warnf("Error calling URL: [%v] for %s : [%v]", url, pc, err.Error())
		return false, false
	}

	defer cleanupResp(resp)

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.IsMarkedDeleted {
		log.Infof("Content Marked deleted. Checking %s, status code [%v]", pc, resp.StatusCode)
		return resp.StatusCode == 404, false
	}

	// if not marked deleted, operation isn't finished until status is 200
	if resp.StatusCode != 200 {
		if resp.StatusCode != 404 {
			log.Infof("Checking %s, status code [%v]", pc, resp.StatusCode)
		}
		return false, false
	}

	// if status is 200, we check the publishReference
	// this way we can handle updates
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Checking %s. Cannot read response: [%s]",
			LoggingContextForCheck(pm.Config.Alias,
				pm.UUID,
				pm.Platform,
				pm.TID),
			err.Error())
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		log.Warnf("Checking %s. Cannot unmarshal JSON response: [%s]",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), err.Error())
		return false, false
	}

	return pm.UUID == jsonResp["uuid"].(string), false
}

// S3Check implements the EndpointSpecificCheck interface to check operation
// status for the S3 endpoint.
type S3Check struct {
	httpCaller httpcaller.Caller
}

func NewS3Check(httpCaller httpcaller.Caller) S3Check {
	return S3Check{httpCaller: httpCaller}
}

// ignoreCheck is always false
func (s S3Check) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	url := pm.Endpoint.String() + pm.UUID
	resp, err := s.httpCaller.DoCall(httpcaller.Config{URL: url}) //nolint:bodyclose
	if err != nil {
		log.Warnf("Checking %s. Error calling URL: [%v] : [%v]", LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), url, err.Error())
		return false, false
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		/*	for S3 files, we're getting a 403 if the files are not yet in, so we're not warning on that */
		if resp.StatusCode != 403 {
			log.Warnf("Checking %s. Error calling URL: [%v] : Response status: [%v]", LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), url, resp.Status)
		}
		return false, false
	}

	// we have to check if the body is null because of an issue where the image is
	// uploaded to S3, but body is empty - in this case, we get 200 back but empty body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Checking %s. Cannot read response: [%s]", LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), err.Error())
		return false, false
	}

	if len(data) == 0 {
		log.Warnf("Checking %s. Image body is empty!", LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID))
		return false, false
	}
	return true, false
}

// NotificationsCheck implements the EndpointSpecificCheck interface to build the endpoint URL and
// to check the operation is present in the notification feed
type NotificationsCheck struct {
	httpCaller      httpcaller.Caller
	subscribedFeeds map[string][]feeds.Feed
	feedName        string
}

func NewNotificationsCheck(httpCaller httpcaller.Caller, subscribedFeeds map[string][]feeds.Feed, feedName string) NotificationsCheck {
	return NotificationsCheck{
		httpCaller:      httpCaller,
		subscribedFeeds: subscribedFeeds,
		feedName:        feedName,
	}
}

func (n NotificationsCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	notifications := n.checkFeed(pc.Metric.UUID, pc.Metric.Platform)
	for _, e := range notifications {
		checkData := map[string]interface{}{"publishReference": e.PublishReference, "lastModified": e.LastModified}
		operationFinished, ignoreCheck := isSamePublishEvent(checkData, pc)
		if operationFinished || ignoreCheck {
			return operationFinished, ignoreCheck
		}
	}

	return false, n.shouldSkipCheck(pc)
}

func (n NotificationsCheck) shouldSkipCheck(pc *PublishCheck) bool {
	pm := pc.Metric
	if !pm.IsMarkedDeleted {
		return false
	}
	url := pm.Endpoint.String() + "/" + pm.UUID
	resp, err := n.httpCaller.DoCall(httpcaller.Config{URL: url, Username: pc.username, Password: pc.password, TxID: httpcaller.ConstructPamTxId(pm.TID)}) //nolint:bodyclose
	if err != nil {
		log.Warnf("Checking %s. Error calling URL: [%v] : [%v]", LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), url, err.Error())
		return false
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		return false
	}

	var notifications []feeds.Notification
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

func (n NotificationsCheck) checkFeed(uuid string, envName string) []*feeds.Notification {
	envFeeds, found := n.subscribedFeeds[envName]
	if found {
		for _, f := range envFeeds {
			if f.FeedName() == n.feedName {
				notifications := f.NotificationsFor(uuid)
				return notifications
			}
		}
	}

	return []*feeds.Notification{}
}

func cleanupResp(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		log.Warnf("[%v]", err)
	}
	err = resp.Body.Close()
	if err != nil {
		log.Warnf("[%v]", err)
	}
}

func isSamePublishEvent(jsonContent map[string]interface{}, pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	if jsonContent["publishReference"] == pm.TID {
		log.Infof("Checking %s. Matched publish reference.", pc)
		return true, false
	}

	// look for rapid-fire publishes
	lastModifiedDate, ok := parseLastModifiedDate(jsonContent)
	if ok {
		if lastModifiedDate.After(pm.PublishDate) {
			log.Infof("Checking %s. Last modified date [%v] is after publish date [%v]", pc, lastModifiedDate, pm.PublishDate)
			return false, true
		}
		if lastModifiedDate.Equal(pm.PublishDate) {
			log.Infof("Checking %s. Last modified date [%v] is equal to publish date [%v]", pc, lastModifiedDate, pm.PublishDate)
			return true, false
		}
		log.Infof("Checking %s. Last modified date [%v] is before publish date [%v]", pc, lastModifiedDate, pm.PublishDate)
	} else {
		log.Warnf("The field 'lastModified' is not valid: [%v]. Skip checking rapid-fire publishes for %s.", jsonContent["lastModified"], pc)
	}

	return false, false
}

func parseLastModifiedDate(jsonContent map[string]interface{}) (*time.Time, bool) {
	lastModifiedDateAsString, ok := jsonContent["lastModified"].(string)
	if ok && lastModifiedDateAsString != "" {
		lastModifiedDate, err := time.Parse(DateLayout, lastModifiedDateAsString)
		return &lastModifiedDate, err == nil
	}
	return nil, false
}

func LoggingContextForCheck(checkType string, uuid string, environment string, transactionID string) string {
	return fmt.Sprintf("environment=[%v], checkType=[%v], uuid=[%v], transaction_id=[%v]", environment, checkType, uuid, transactionID)
}
