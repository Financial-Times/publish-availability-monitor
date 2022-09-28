package checks

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
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
	log                    *logger.UPPLogger
}

// NewPublishCheck returns a PublishCheck ready to perform a check for pm.UUID, at the pm.Endpoint.
func NewPublishCheck(
	metric metrics.PublishMetric,
	username, password string,
	threshold, checkInterval int,
	resultSink chan metrics.PublishMetric,
	endpointSpecificChecks map[string]EndpointSpecificCheck,
	log *logger.UPPLogger,
) *PublishCheck {
	return &PublishCheck{
		Metric:                 metric,
		username:               username,
		password:               password,
		Threshold:              threshold,
		CheckInterval:          checkInterval,
		ResultSink:             resultSink,
		endpointSpecificChecks: endpointSpecificChecks,
		log:                    log,
	}
}

// DoCheck performs an availability check on a piece of content at a certain
// endpoint, applying endpoint-specific processing.
// Returns true if the content is available at the endpoint, false otherwise.
func (pc PublishCheck) DoCheck() (checkSuccessful, ignoreCheck bool) {
	pc.log.Infof("Running check for %s\n", pc)
	check := pc.endpointSpecificChecks[pc.Metric.Config.Alias]
	if check == nil {
		pc.log.Warnf("No check for %s", pc)
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
	resp, err := c.httpCaller.DoCall(httpcaller.Config{
		URL:      url,
		Username: pc.username,
		Password: pc.password,
		TID:      httpcaller.ConstructPamTID(pm.TID),
	})
	if err != nil {
		pc.log.WithError(err).Warnf("Error calling URL: [%v] for %s", url, pc)
		return false, false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.IsMarkedDeleted {
		pc.log.Infof("Content Marked deleted. Checking %s, status code [%v]", pc, resp.StatusCode)
		return resp.StatusCode == 404, false
	}

	// if not marked deleted, operation isn't finished until status is 200
	if resp.StatusCode != 200 {
		if resp.StatusCode != 404 {
			pc.log.Infof("Checking %s, status code [%v]", pc, resp.StatusCode)
		}
		return false, false
	}

	// if status is 200, we check the publishReference
	// this way we can handle updates
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		pc.log.WithError(err).Warnf("Checking %s. Cannot read response",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID))
		return false, false
	}

	var jsonResp map[string]interface{}

	if err = json.Unmarshal(data, &jsonResp); err != nil {
		pc.log.WithError(err).Warnf("Checking %s. Cannot unmarshal JSON response",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID))
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

//nolint:unparam
func (c ContentNeo4jCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	url := pm.Endpoint.String() + pm.UUID

	resp, err := c.httpCaller.DoCall(httpcaller.Config{
		URL:      url,
		Username: pc.username,
		Password: pc.password,
		TID:      httpcaller.ConstructPamTID(pm.TID)})

	if err != nil {
		pc.log.Warnf("Error calling URL: [%v] for %s", url, pc)
		return false, false
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.IsMarkedDeleted {
		pc.log.Infof("Content Marked deleted. Checking %s, status code [%v]", pc, resp.StatusCode)
		return resp.StatusCode == 404, false
	}

	// if not marked deleted, operation isn't finished until status is 200
	if resp.StatusCode != 200 {
		if resp.StatusCode != 404 {
			pc.log.Infof("Checking %s, status code [%v]", pc, resp.StatusCode)
		}
		return false, false
	}

	// if status is 200, we check the publishReference
	// this way we can handle updates
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		pc.log.WithError(err).Warnf("Checking %s. Cannot read response",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID))
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		pc.log.WithError(err).Warnf("Checking %s. Cannot unmarshal JSON response",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID))
		return false, false
	}

	return pm.UUID == jsonResp["uuid"].(string), false
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
	resp, err := n.httpCaller.DoCall(httpcaller.Config{URL: url, Username: pc.username, Password: pc.password, TID: httpcaller.ConstructPamTID(pm.TID)})
	if err != nil {
		pc.log.WithError(err).Warnf("Checking %s. Error calling URL: [%v]",
			LoggingContextForCheck(pm.Config.Alias, pm.UUID, pm.Platform, pm.TID), url)
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

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

func isSamePublishEvent(jsonContent map[string]interface{}, pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	if jsonContent["publishReference"] == pm.TID {
		pc.log.Infof("Checking %s. Matched publish reference.", pc)
		return true, false
	}

	// look for rapid-fire publishes
	lastModifiedDate, ok := parseLastModifiedDate(jsonContent)
	if ok {
		if lastModifiedDate.After(pm.PublishDate) {
			pc.log.Infof("Checking %s. Last modified date [%v] is after publish date [%v]", pc, lastModifiedDate, pm.PublishDate)
			return false, true
		}
		if lastModifiedDate.Equal(pm.PublishDate) {
			pc.log.Infof("Checking %s. Last modified date [%v] is equal to publish date [%v]", pc, lastModifiedDate, pm.PublishDate)
			return true, false
		}
		pc.log.Infof("Checking %s. Last modified date [%v] is before publish date [%v]", pc, lastModifiedDate, pm.PublishDate)
	} else {
		pc.log.Warnf("The field 'lastModified' is not valid: [%v]. Skip checking rapid-fire publishes for %s.", jsonContent["lastModified"], pc)
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
