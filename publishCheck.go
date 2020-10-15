package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/checks"
	"github.com/Financial-Times/publish-availability-monitor/v2/feeds"
	log "github.com/sirupsen/logrus"
)

// PublishCheck performs an availability  check on a piece of content, at a
// given endpoint, and returns whether the check was successful or not.
// Holds all the information necessary to check content availability
// at an endpoint, as well as store and send the results of the check.
type PublishCheck struct {
	Metric        PublishMetric
	username      string
	password      string
	Threshold     int
	CheckInterval int
	ResultSink    chan PublishMetric
}

// EndpointSpecificCheck is the interface which determines the state of the operation we are currently checking.
type EndpointSpecificCheck interface {
	// Returns the state of the operation and whether this check should be ignored
	isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool)
}

// ContentCheck implements the EndpointSpecificCheck interface to check operation
// status for the content endpoint.
type ContentCheck struct {
	httpCaller checks.HttpCaller
}

// ContentCheck implements the EndpointSpecificCheck interface to check operation
// status for the content endpoint.
type ContentNeo4jCheck struct {
	httpCaller checks.HttpCaller
}

func (c ContentNeo4jCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {

	pm := pc.Metric
	url := pm.endpoint.String() + pm.UUID

	resp, err := c.httpCaller.DoCall(checks.Config{
		Url:      url,
		Username: pc.username,
		Password: pc.password,
		TxId:     checks.ConstructPamTxId(pm.tid)})

	if err != nil {
		log.Warnf("Error calling URL: [%v] for %s : [%v]", url, pc, err.Error())
		return false, false
	}

	defer cleanupResp(resp)

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.isMarkedDeleted {
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
			loggingContextForCheck(pm.config.Alias,
				pm.UUID,
				pm.platform,
				pm.tid),
			err.Error())
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		log.Warnf("Checking %s. Cannot unmarshal JSON response: [%s]",
			loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), err.Error())
		return false, false
	}

	return pm.UUID == jsonResp["uuid"].(string), false
}

// S3Check implements the EndpointSpecificCheck interface to check operation
// status for the S3 endpoint.
type S3Check struct {
	httpCaller checks.HttpCaller
}

// NotificationsCheck implements the EndpointSpecificCheck interface to build the endpoint URL and
// to check the operation is present in the notification feed
type NotificationsCheck struct {
	httpCaller      checks.HttpCaller
	subscribedFeeds map[string][]feeds.Feed
	feedName        string
}

// NewPublishCheck returns a PublishCheck ready to perform a check for pm.UUID, at the
// pm.endpoint.
func NewPublishCheck(pm PublishMetric, username string, password string, t int, ci int, rs chan PublishMetric) *PublishCheck {
	return &PublishCheck{pm, username, password, t, ci, rs}
}

var endpointSpecificChecks map[string]EndpointSpecificCheck

func init() {
	hC := checks.NewHttpCaller(10)

	//key is the endpoint alias from the config
	endpointSpecificChecks = map[string]EndpointSpecificCheck{
		"content":                 ContentCheck{hC},
		"content-neo4j":           ContentNeo4jCheck{hC},
		"complementary-content":   ContentCheck{hC},
		"internal-components":     ContentCheck{hC},
		"S3":                      S3Check{hC},
		"enrichedContent":         ContentCheck{hC},
		"lists":                   ContentCheck{hC},
		"notifications":           NotificationsCheck{hC, subscribedFeeds, "notifications"},
		"notifications-push":      NotificationsCheck{hC, subscribedFeeds, "notifications-push"},
		"list-notifications":      NotificationsCheck{hC, subscribedFeeds, "list-notifications"},
		"list-notifications-push": NotificationsCheck{hC, subscribedFeeds, "list-notifications-push"},
	}
}

// DoCheck performs an availability check on a piece of content at a certain
// endpoint, applying endpoint-specific processing.
// Returns true if the content is available at the endpoint, false otherwise.
func (pc PublishCheck) DoCheck() (checkSuccessful, ignoreCheck bool) {
	log.Infof("Running check for %s\n", pc)
	check := endpointSpecificChecks[pc.Metric.config.Alias]
	if check == nil {
		log.Warnf("No check for %s", pc)
		return false, false
	}

	return check.isCurrentOperationFinished(&pc)
}

func (pc PublishCheck) String() string {
	return loggingContextForCheck(pc.Metric.config.Alias, pc.Metric.UUID, pc.Metric.platform, pc.Metric.tid)
}

func (c ContentCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	url := pm.endpoint.String() + pm.UUID
	resp, err := c.httpCaller.DoCall(checks.Config{Url: url, Username: pc.username, Password: pc.password, TxId: checks.ConstructPamTxId(pm.tid)})
	if err != nil {
		log.Warnf("Error calling URL: [%v] for %s : [%v]", url, pc, err.Error())
		return false, false
	}
	defer cleanupResp(resp)

	// if the article was marked as deleted, operation is finished when the
	// article cannot be found anymore
	if pm.isMarkedDeleted {
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
			loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), err.Error())
		return false, false
	}

	var jsonResp map[string]interface{}

	err = json.Unmarshal(data, &jsonResp)
	if err != nil {
		log.Warnf("Checking %s. Cannot unmarshal JSON response: [%s]",
			loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), err.Error())
		return false, false
	}

	return isSamePublishEvent(jsonResp, pc)
}

func isSamePublishEvent(jsonContent map[string]interface{}, pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	if jsonContent["publishReference"] == pm.tid {
		log.Infof("Checking %s. Matched publish reference.", pc)
		return true, false
	}

	// look for rapid-fire publishes
	lastModifiedDate, ok := parseLastModifiedDate(jsonContent)
	if ok {
		if (*lastModifiedDate).After(pm.publishDate) {
			log.Infof("Checking %s. Last modified date [%v] is after publish date [%v]", pc, lastModifiedDate, pm.publishDate)
			return false, true
		}
		if (*lastModifiedDate).Equal(pm.publishDate) {
			log.Infof("Checking %s. Last modified date [%v] is equal to publish date [%v]", pc, lastModifiedDate, pm.publishDate)
			return true, false
		}
		log.Infof("Checking %s. Last modified date [%v] is before publish date [%v]", pc, lastModifiedDate, pm.publishDate)
	} else {
		log.Warnf("The field 'lastModified' is not valid: [%v]. Skip checking rapid-fire publishes for %s.", jsonContent["lastModified"], pc)
	}

	return false, false
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
func (s S3Check) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	pm := pc.Metric
	url := pm.endpoint.String() + pm.UUID
	resp, err := s.httpCaller.DoCall(checks.Config{Url: url})
	if err != nil {
		log.Warnf("Checking %s. Error calling URL: [%v] : [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), url, err.Error())
		return false, false
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		/*	for S3 files, we're getting a 403 if the files are not yet in, so we're not warning on that */
		if resp.StatusCode != 403 {
			log.Warnf("Checking %s. Error calling URL: [%v] : Response status: [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), url, resp.Status)
		}
		return false, false
	}

	// we have to check if the body is null because of an issue where the image is
	// uploaded to S3, but body is empty - in this case, we get 200 back but empty body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Checking %s. Cannot read response: [%s]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), err.Error())
		return false, false
	}

	if len(data) == 0 {
		log.Warnf("Checking %s. Image body is empty!", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid))
		return false, false
	}
	return true, false
}

func (n NotificationsCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished, ignoreCheck bool) {
	notifications := n.checkFeed(pc.Metric.UUID, pc.Metric.platform)
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
	if !pm.isMarkedDeleted {
		return false
	}
	url := pm.endpoint.String() + "/" + pm.UUID
	resp, err := n.httpCaller.DoCall(checks.Config{Url: url, Username: pc.username, Password: pc.password, TxId: checks.ConstructPamTxId(pm.tid)})
	if err != nil {
		log.Warnf("Checking %s. Error calling URL: [%v] : [%v]", loggingContextForCheck(pm.config.Alias, pm.UUID, pm.platform, pm.tid), url, err.Error())
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
