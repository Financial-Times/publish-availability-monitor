package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	"github.com/Financial-Times/service-status-go/gtg"
)

const requestTimeout = 4500

// Healthcheck offers methods to measure application health.
type Healthcheck struct {
	client          *http.Client
	config          *config.AppConfig
	consumer        kafkaConsumer
	metricContainer *metrics.History
	environments    *envs.Environments
	subscribedFeeds map[string][]feeds.Feed
	log             *logger.UPPLogger
}

type kafkaConsumer interface {
	ConnectivityCheck() error
	MonitorCheck() error
}

func newHealthcheck(config *config.AppConfig, metricContainer *metrics.History, environments *envs.Environments, subscribedFeeds map[string][]feeds.Feed, c kafkaConsumer, log *logger.UPPLogger) *Healthcheck {
	httpClient := &http.Client{Timeout: requestTimeout * time.Millisecond}
	return &Healthcheck{
		client:          httpClient,
		config:          config,
		consumer:        c,
		metricContainer: metricContainer,
		environments:    environments,
		subscribedFeeds: subscribedFeeds,
		log:             log,
	}
}

type readEnvironmentHealthcheck struct {
	env       envs.Environment
	client    *http.Client
	appConfig *config.AppConfig
	log       *logger.UPPLogger
}

const pamRunbookURL = "https://runbooks.in.ft.com/publish-availability-monitor"

var noReadEnvironments = fthealth.Check{
	ID:               "ReadEnvironments",
	BusinessImpact:   "Publish metrics are not recorded. This will impact the SLA measurement.",
	Name:             "ReadEnvironments",
	PanicGuide:       pamRunbookURL,
	Severity:         1,
	TechnicalSummary: "There are no read environments to monitor. This could be because none have been configured",
	Checker: func() (string, error) {
		return "", errors.New("there are no read environments to monitor")
	},
}

func (h *Healthcheck) checkHealth() func(w http.ResponseWriter, r *http.Request) {
	checks := []fthealth.Check{
		h.consumerQueueReachable(),
		h.reflectPublishFailures(),
		h.validationServicesReachable(),
		h.isConsumingFromPushFeeds(),
		h.consumerMonitorCheck(),
	}

	readEnvironmentChecks := h.readEnvironmentsReachable()
	if len(readEnvironmentChecks) == 0 {
		checks = append(checks, noReadEnvironments)
	} else {
		checks = append(checks, readEnvironmentChecks...)
	}

	hc := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  "publish-availability-monitor",
			Name:        "Publish Availability Monitor",
			Description: "Monitors publishes to the UPP platform and alerts on any publishing failures",
			Checks:      checks,
		},
		Timeout: 10 * time.Second,
	}

	return fthealth.Handler(hc)
}

func (h *Healthcheck) GTG() gtg.Status {
	consumerCheck := func() gtg.Status {
		return gtgCheck(h.checkConsumerConnectivity)
	}

	validationServiceCheck := func() gtg.Status {
		return gtgCheck(h.checkValidationServicesReachable)
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{
		consumerCheck,
		validationServiceCheck,
	})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *Healthcheck) isConsumingFromPushFeeds() fthealth.Check {
	return fthealth.Check{
		ID:               "IsConsumingFromNotificationsPushFeeds",
		BusinessImpact:   "Publish metrics are not recorded. This will impact the SLA measurement.",
		Name:             "IsConsumingFromNotificationsPushFeeds",
		PanicGuide:       pamRunbookURL,
		Severity:         1,
		TechnicalSummary: "The connections to the configured notifications-push feeds are operating correctly.",
		Checker: func() (string, error) {
			var failing []string
			result := true
			for _, val := range h.subscribedFeeds {
				for _, feed := range val {
					push, ok := feed.(*feeds.NotificationsPushFeed)
					if ok && !push.IsConnected() {
						h.log.Warnf("Feed \"%s\" with URL \"%s\" is not connected!", feed.FeedName(), feed.FeedURL())
						failing = append(failing, feed.FeedURL())
						result = false
					}
				}
			}

			if !result {
				return "Disconnection detected.", errors.New("At least one of our Notifications Push feeds in the delivery cluster is disconnected! " +
					"Please review the logs, and check delivery healthchecks. " +
					"We will attempt reconnection indefinitely, but there could be an issue with the delivery cluster's notifications-push services. " +
					"Failing connections: " + strings.Join(failing, ","))
			}
			return "", nil
		},
	}
}

func (h *Healthcheck) consumerQueueReachable() fthealth.Check {
	return fthealth.Check{
		ID:               "ConsumerQueueReachable",
		BusinessImpact:   "Publish metrics are not recorded. This will impact the SLA measurement.",
		Name:             "ConsumerQueueReachable",
		PanicGuide:       pamRunbookURL,
		Severity:         1,
		TechnicalSummary: "Kafka consumer is not reachable/healthy",
		Checker:          h.checkConsumerConnectivity,
	}
}

func (h *Healthcheck) consumerMonitorCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "ConsumerQueueLagging",
		BusinessImpact:   "Publish metrics are slowed down. This will impact the SLA measurement.",
		Name:             "ConsumerQueueLagging",
		PanicGuide:       pamRunbookURL,
		Severity:         2,
		TechnicalSummary: kafka.LagTechnicalSummary,
		Checker:          h.checkConsumerMonitor,
	}
}

func (h *Healthcheck) reflectPublishFailures() fthealth.Check {
	return fthealth.Check{
		ID:               "ReflectPublishFailures",
		BusinessImpact:   "At least two of the last 10 publishes failed. This will reflect in the SLA measurement.",
		Name:             "ReflectPublishFailures",
		PanicGuide:       pamRunbookURL,
		Severity:         1,
		TechnicalSummary: "Publishes did not meet the SLA measurments",
		Checker:          h.checkForPublishFailures,
	}
}

func (h *Healthcheck) checkForPublishFailures() (string, error) {
	failures := h.metricContainer.GetFailures()

	failureThreshold := 2 //default
	if h.config.HealthConf.FailureThreshold != 0 {
		failureThreshold = h.config.HealthConf.FailureThreshold
	}

	if len(failures) >= failureThreshold {
		return "", fmt.Errorf("%d publish failures happened during the last 10 publishes", len(failures))
	}
	return "", nil
}

func (h *Healthcheck) validationServicesReachable() fthealth.Check {
	return fthealth.Check{
		ID:               "validationServicesReachable",
		BusinessImpact:   "Publish metrics might not be correct. False positive failures might be recorded. This will impact the SLA measurement.",
		Name:             "validationServicesReachable",
		PanicGuide:       pamRunbookURL,
		Severity:         1,
		TechnicalSummary: "Validation services are not reachable/healthy",
		Checker:          h.checkValidationServicesReachable,
	}
}

func (h *Healthcheck) checkValidationServicesReachable() (string, error) {
	endpoints := h.config.ValidationEndpoints
	var wg sync.WaitGroup
	hcErrs := make(chan error, len(endpoints))
	for _, url := range endpoints {
		wg.Add(1)
		healthcheckURL, err := inferHealthCheckURL(url)
		if err != nil {
			h.log.WithError(err).Errorf("Validation Service URL: [%s].", url)
			continue
		}
		username, password := envs.GetValidationCredentials()
		go checkServiceReachable(healthcheckURL, username, password, h.client, hcErrs, &wg, h.log)
	}

	wg.Wait()
	close(hcErrs)
	for err := range hcErrs {
		if err != nil {
			return "", err
		}
	}
	return "", nil
}

func (h *Healthcheck) checkConsumerConnectivity() (string, error) {
	if err := h.consumer.ConnectivityCheck(); err != nil {
		return "", err
	}
	return "OK", nil
}

func (h *Healthcheck) checkConsumerMonitor() (string, error) {
	if err := h.consumer.MonitorCheck(); err != nil {
		return "", err
	}
	return "OK", nil
}

func checkServiceReachable(healthcheckURL string, username string, password string, client *http.Client, hcRes chan<- error, wg *sync.WaitGroup, log *logger.UPPLogger) {
	defer wg.Done()
	log.Debugf("Checking: %s", healthcheckURL)

	req, err := http.NewRequest("GET", healthcheckURL, nil)
	if err != nil {
		hcRes <- fmt.Errorf("cannot create HTTP request with URL: [%s]: [%w]", healthcheckURL, err)
		return
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		hcRes <- fmt.Errorf("healthcheck URL: [%s]: [%w]", healthcheckURL, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		hcRes <- fmt.Errorf("unhealthy statusCode received: [%d] for URL [%s]", resp.StatusCode, healthcheckURL)
		return
	}
	hcRes <- nil
}

func (h *Healthcheck) readEnvironmentsReachable() []fthealth.Check {
	for i := 0; !h.environments.AreReady() && i < 5; i++ {
		h.log.Info("Environments not set, retry in 2s...")
		time.Sleep(2 * time.Second)
	}

	hc := make([]fthealth.Check, 0, h.environments.Len())

	for _, envName := range h.environments.Names() {
		hc = append(hc, fthealth.Check{
			ID:               envName + "-readEndpointsReachable",
			BusinessImpact:   "Publish metrics might not be correct. False positive failures might be recorded. This will impact the SLA measurement.",
			Name:             envName + "-readEndpointsReachable",
			PanicGuide:       pamRunbookURL,
			Severity:         1,
			TechnicalSummary: "Read services are not reachable/healthy",
			Checker: (&readEnvironmentHealthcheck{
				env:       h.environments.Environment(envName),
				client:    h.client,
				appConfig: h.config,
				log:       h.log,
			}).checkReadEnvironmentReachable,
		})
	}
	return hc
}

func (h *readEnvironmentHealthcheck) checkReadEnvironmentReachable() (string, error) {
	var wg sync.WaitGroup
	hcErrs := make(chan error, len(h.appConfig.MetricConf))

	for _, metric := range h.appConfig.MetricConf {
		var endpointURL *url.URL
		var err error
		var username, password string
		if checks.AbsoluteURLRegex.MatchString(metric.Endpoint) {
			endpointURL, err = url.Parse(metric.Endpoint)
		} else {
			endpointURL, err = url.Parse(h.env.ReadURL + metric.Endpoint)
			username = h.env.Username
			password = h.env.Password
		}

		if err != nil {
			h.log.WithError(err).Errorf("Cannot parse url [%v]", metric.Endpoint)
			continue
		}

		healthcheckURL := buildFtHealthcheckURL(*endpointURL, metric.Health)

		wg.Add(1)
		go checkServiceReachable(healthcheckURL, username, password, h.client, hcErrs, &wg, h.log)
	}

	wg.Wait()
	close(hcErrs)
	for err := range hcErrs {
		if err != nil {
			return "", err
		}
	}
	return "", nil
}

func inferHealthCheckURL(serviceURL string) (string, error) {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return "", err
	}

	var newPath string
	if strings.HasPrefix(parsedURL.Path, "/__") {
		newPath = strings.SplitN(parsedURL.Path[1:], "/", 2)[0] + "/__health"
	} else {
		newPath = "/__health"
	}

	parsedURL.Path = newPath
	return parsedURL.String(), nil
}

func buildFtHealthcheckURL(endpoint url.URL, health string) string {
	endpoint.Path = health
	endpoint.RawQuery = "" // strip query params
	return endpoint.String()
}
