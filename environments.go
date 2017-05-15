package main

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"net/url"
	"os"
	"time"
)

func watchConfigFiles(envsFileName string, envCredentialsFileName string, validationCredentialsFileName string, configRefreshPeriod int) {
	ticker := time.NewTicker(time.Minute * configRefreshPeriod)

	for range ticker.C {
		err := updateEnvsAndValidationCredentials(envsFileName, envCredentialsFileName, validationCredentialsFileName)
		if err != nil {
			errorLogger.Printf("Cannot update configuration, error was: %s", err)
		}
	}
}

func updateEnvsAndValidationCredentials(envsFileName string, envCredentialsFileName string, validationCredentialsFileName string) error {
	err := updateEnvs(envsFileName, envCredentialsFileName)
	if err != nil {
		return fmt.Errorf("Cannot update envs. Error was: %s", err)
	}

	err = updateValidationCredentials(validationCredentialsFileName)
	if err != nil {
		return fmt.Errorf("Cannot update envs. Error was: %s", err)
	}

	return nil
}

func updateEnvs(envsFileName string, envCredentialsFileName string) error {
	//todo: remove this log
	infoLogger.Print("Updating envs")

	envsFromFile, err := readEnvs(envsFileName)
	if err != nil {
		return fmt.Errorf("Cannot parse environments. Error was: %s", err)
	}

	validEnvs := filterInvalidEnvs(envsFromFile)

	envCredentials, err := readEnvCredentials(envCredentialsFileName)
	if err != nil {
		return fmt.Errorf("Cannot parse environments. Error was: %s", err)
	}

	removedEnvs := parseEnvsIntoMap(validEnvs, envCredentials)
	configureFeeds(removedEnvs)

	return nil
}

func updateValidationCredentials(validationCredsFileName string) error {
	//todo: remove this log
	infoLogger.Print("Updating validation credentials")
	data, err := os.Open(validationCredsFileName)
	if err != nil {
		return err
	}

	jsonParser := json.NewDecoder(data)
	credentials := Credentials{}
	err = jsonParser.Decode(&credentials)
	if err != nil {
		return err
	}

	validatorCredentials = credentials
	return nil
}

func configureFeeds(removedEnvs []string) {
	for _, envName := range removedEnvs {
		feeds, found := subscribedFeeds[envName]
		if found {
			for _, f := range feeds {
				f.Stop()
			}
		}

		delete(subscribedFeeds, envName)
	}

	for _, metric := range appConfig.MetricConf {
		for _, env := range environments {
			var envFeeds []feeds.Feed
			var found bool
			if envFeeds, found = subscribedFeeds[env.Name]; !found {
				envFeeds = make([]feeds.Feed, 0)
			}

			found = false
			for _, f := range envFeeds {
				if f.FeedName() == metric.Alias {
					f.SetCredentials(env.Username, env.Password)
					found = true
					break
				}
			}

			if !found {
				endpointUrl, err := url.Parse(env.ReadUrl + metric.Endpoint)
				if err != nil {
					errorLogger.Printf("Cannot parse url [%v], error: [%v]", metric.Endpoint, err.Error())
					continue
				}

				interval := appConfig.Threshold / metric.Granularity

				if f := feeds.NewNotificationsFeed(metric.Alias, *endpointUrl, appConfig.Threshold, interval, env.Username, env.Password); f != nil {
					subscribedFeeds[env.Name] = append(envFeeds, f)
					f.Start()
				}
			}
		}
	}
}

func filterInvalidEnvs(envsFromFile []Environment) []Environment {
	var validEnvs []Environment
	for _, env := range envsFromFile {
		//envs without name are invalid
		if env.Name == "" {
			errorLogger.Printf("Env %v has an empty name, skipping it", env)
			continue
		}

		//envs without read-url are invalid
		if env.ReadUrl == "" {
			errorLogger.Printf("Env with name %s does not have readUrl, skipping it", env.Name)
			continue
		}

		//envs without s3 are still valid, but still a heads up is given.
		if env.S3Url == "" {
			infoLogger.Printf("Env with name %s does not have s3 url.", env.S3Url)
		}

		validEnvs = append(validEnvs, env)
	}

	return validEnvs
}

func parseEnvsIntoMap(envs []Environment, envCredentials []Credentials) []string {
	//enhance envs with credentials
	for i, env := range envs {
		for _, envCredentials := range envCredentials {
			if env.Name == envCredentials.EnvName {
				envs[i].Username = envCredentials.Username
				envs[i].Password = envCredentials.Password
				break
			}
		}

		if envs[i].Username == "" || envs[i].Password == "" {
			infoLogger.Printf("No credentials provided for env with name %s", env.Name)
		}
	}

	//remove envs that don't exist anymore
	removedEnvs := make([]string, 0)
	for envName := range environments {
		if !isEnvInSlice(envName, envs) {
			infoLogger.Printf("removing environment from monitoring: %v", envName)
			delete(environments, envName)
			removedEnvs = append(removedEnvs, envName)
		}
	}

	//update envs
	for _, env := range envs {
		envName := env.Name
		environments[envName] = env
		infoLogger.Printf("Added environment to monitoring: %s", envName)
	}

	return removedEnvs
}

func readEnvs(fileName string) ([]Environment, error) {
	data, err := os.Open(fileName)
	if err != nil {
		return []Environment{}, err
	}

	jsonParser := json.NewDecoder(data)
	envs := []Environment{}
	err = jsonParser.Decode(&envs)
	return envs, err
}

func readEnvCredentials(fileName string) ([]Credentials, error) {
	data, err := os.Open(fileName)
	if err != nil {
		return []Credentials{}, err
	}

	jsonParser := json.NewDecoder(data)
	credentials := []Credentials{}
	err = jsonParser.Decode(&credentials)

	return credentials, err
}

func isEnvInSlice(envName string, envs []Environment) bool {
	for _, env := range envs {
		if env.Name == envName {
			return true
		}
	}

	return false
}
