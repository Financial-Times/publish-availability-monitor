package main

import "net/url"

func runGtgChecks(allEnvironments map[string]Environment, checkedEnvironment Environment) bool {
	gtg := endpointSpecificChecks["gtg"]
	check := generateGtgCheckForEnvironment(checkedEnvironment)
	if check == nil {
		return false
	}

	_, ignore := gtg.isCurrentOperationFinished(check)
	if ignore { // This environment is failing, and could be ignored if other environments are ok.
		for name, environment := range allEnvironments {
			if checkedEnvironment.Name == name {
				continue
			}

			_, unhealthy := gtg.isCurrentOperationFinished(generateGtgCheckForEnvironment(environment))
			if !unhealthy { // If the other environment is healthy then we can ignore this failure
				return ignore
			}
		}
	}

	return false // This environment is healthy, OR no environments are healthy, so do not ignore failure
}

func generateGtgCheckForEnvironment(env Environment) *PublishCheck {
	endpoint, err := url.Parse(env.ReadUrl + "/__gtg")
	if err != nil {
		errorLogger.Printf("Failed to generate gtg endpoint from read url! url: [%s] - [%v]", env.ReadUrl, err.Error())
		return nil
	}

	metric := &PublishMetric {
		endpoint: *endpoint,
	}

	return &PublishCheck {
		username: env.Username,
		password: env.Password,
		Metric: *metric,
	}
}

func (c GTGCheck) isCurrentOperationFinished(pc *PublishCheck) (operationFinished bool, clusterUnhealthy bool) {
	infoLogger.Printf("Checking GTG endpoint for cluster [%s]", pc.Metric.endpoint)

	resp, err := c.httpCaller.doCall(pc.Metric.endpoint.String(), pc.username, pc.password)

	if err != nil {
		warnLogger.Printf("Error calling GTG URL: [%v] for %s : [%v]", pc.Metric.endpoint, pc, err.Error())
		return true, true
	}

	if resp.StatusCode != 200 {
		infoLogger.Printf("GTG Url for delivery cluster [%s] is unhealthy!", pc.Metric.endpoint.String())
		return true, true
	}

	infoLogger.Printf("Delivery cluster is healthy. [%s]", pc.Metric.endpoint.String())
	return true, false
}