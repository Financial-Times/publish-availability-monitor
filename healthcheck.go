package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Financial-Times/go-fthealth"
)

// Healthcheck offers methods to measure application health.
type Healthcheck struct {
	client          http.Client
	config          AppConfig
	metricContainer *publishHistory
}

func (h *Healthcheck) checkHealth() func(w http.ResponseWriter, r *http.Request) {
	return fthealth.HandlerParallel(
		"Dependent services healthcheck", "Checks if all the dependent services are reachable and healthy.",
		h.messageQueueProxyReachable(),
		h.reflectPublishFailures(),
		h.validationServicesReachable(),
	)
}

func (h *Healthcheck) gtg(writer http.ResponseWriter, req *http.Request) {
	healthChecks := []func() error{h.checkAggregateMessageQueueProxiesReachable}

	for _, hCheck := range healthChecks {
		if err := hCheck(); err != nil {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
}

func (h *Healthcheck) messageQueueProxyReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Publish metrics are not recorded. This will impact the SLA measurement.",
		Name:             "MessageQueueProxyReachable",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/publish-availability-monitor-run-book",
		Severity:         1,
		TechnicalSummary: "Message queue proxy is not reachable/healthy",
		Checker:          h.checkAggregateMessageQueueProxiesReachable,
	}

}

func (h *Healthcheck) checkAggregateMessageQueueProxiesReachable() error {

	addresses := h.config.QueueConf.Addrs
	errMsg := ""
	for i := 0; i < len(addresses); i++ {
		error := h.checkMessageQueueProxyReachable(addresses[i])
		if error == nil {
			return nil
		}
		errMsg = errMsg + fmt.Sprintf("For %s there is an error %v \n", addresses[i], error.Error())
	}

	return errors.New(errMsg)

}

func (h *Healthcheck) checkMessageQueueProxyReachable(address string) error {
	req, err := http.NewRequest("GET", address+"/topics", nil)
	if err != nil {
		warnLogger.Printf("Could not connect to proxy: %v", err.Error())
		return err
	}

	if len(h.config.QueueConf.AuthorizationKey) > 0 {
		req.Header.Add("Authorization", h.config.QueueConf.AuthorizationKey)
	}

	if len(h.config.QueueConf.Queue) > 0 {
		req.Host = h.config.QueueConf.Queue
	}

	resp, err := h.client.Do(req)
	if err != nil {
		warnLogger.Printf("Could not connect to proxy: %v", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Proxy returned status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}

	body, err := ioutil.ReadAll(resp.Body)
	return checkIfTopicIsPresent(body, h.config.QueueConf.Topic)

}

func checkIfTopicIsPresent(body []byte, searchedTopic string) error {
	var topics []string

	err := json.Unmarshal(body, &topics)
	if err != nil {
		return fmt.Errorf("Error occured and topic could not be found. %v", err.Error())
	}

	for _, topic := range topics {
		if topic == searchedTopic {
			return nil
		}
	}

	return errors.New("Topic was not found")
}

func (h *Healthcheck) reflectPublishFailures() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "At least two of the last 10 publishes failed. This will reflect in the SLA measurement.",
		Name:             "ReflectPublishFailures",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/publish-availability-monitor-run-book",
		Severity:         1,
		TechnicalSummary: "Publishes did not meet the SLA measurments",
		Checker:          h.checkForPublishFailures,
	}

}

func (h *Healthcheck) checkForPublishFailures() error {
	metricContainer.RLock()
	failures := 0
	for i := 0; i < len(metricContainer.publishMetrics); i++ {
		if !metricContainer.publishMetrics[i].publishOK {
			failures++
		}
	}
	metricContainer.RUnlock()

	failureThreshold := 2 //default
	if h.config.HealthConf.FailureThreshold != 0 {
		failureThreshold = h.config.HealthConf.FailureThreshold
	}
	if failures >= failureThreshold {
		return fmt.Errorf("%d publish failures happened during the last 10 publishes", failures)
	}
	return nil
}

func (h *Healthcheck) validationServicesReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Publish metrics might not be correct. False positive failures might be recorded. This will impact the SLA measurement.",
		Name:             "validationServicesReachable",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/publish-availability-monitor-run-book",
		Severity:         1,
		TechnicalSummary: "Message queue proxy is not reachable/healthy",
		Checker:          h.checkValidationServicesReachable,
	}
}

func (h *Healthcheck) checkValidationServicesReachable() error {
	endpoints := h.config.ValidationEndpoints
	var wg sync.WaitGroup
	hcErrs := make(chan error, len(endpoints))
	for _, url := range endpoints {
		wg.Add(1)
		go checkValidationServiceReachable(url, hcErrs, &wg)
	}
	wg.Wait()
	close(hcErrs)
	for err := range hcErrs {
		if err != nil {
			return err
		}
	}
	return nil
}

func checkValidationServiceReachable(validationURL string, hcRes chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	healthURL, err := buildHealthURL(validationURL)
	if err != nil {
		hcRes <- fmt.Errorf("Validation URL: [%s]. Error: [%v]", validationURL, err)
		return
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		hcRes <- fmt.Errorf("Healthcheck URL: [%s]. Error: [%v]", healthURL, err)
		return
	}
	defer cleanupResp(resp)
	hcRes <- nil
}

func buildHealthURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsedURL.Path = trimRightmostSubpath(parsedURL.Path) + "/__health"
	return parsedURL.String(), nil
}

func trimRightmostSubpath(urlPath string) string {
	res := strings.TrimRight(urlPath, "/")
	slashI := strings.LastIndex(res, "/")
	if slashI != -1 {
		res = res[:slashI]
	}
	return res
}
