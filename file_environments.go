package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"sync"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/feeds"
	log "github.com/sirupsen/logrus"
)

func watchConfigFiles(wg *sync.WaitGroup, envsFileName string, envCredentialsFileName string, validationCredentialsFileName string, configRefreshPeriod int) {
	ticker := newTicker(0, time.Minute*time.Duration(configRefreshPeriod))
	first := true
	defer func() {
		markWaitGroupDone(wg, first)
	}()

	for range ticker.C {
		err := updateEnvsIfChanged(envsFileName, envCredentialsFileName)
		if err != nil {
			log.Errorf("Could not update envs config, error was: %s", err)
		}

		err = updateValidationCredentialsIfChanged(validationCredentialsFileName)
		if err != nil {
			log.Errorf("Could not update validation credentials config, error was: %s", err)
		}

		first = markWaitGroupDone(wg, first)
	}
}

func markWaitGroupDone(wg *sync.WaitGroup, first bool) bool {
	if first {
		wg.Done()
		first = false
	}

	return first
}

func newTicker(delay, repeat time.Duration) *time.Ticker {
	// adapted from https://stackoverflow.com/questions/32705582/how-to-get-time-tick-to-tick-immediately
	ticker := time.NewTicker(repeat)
	oc := ticker.C
	nc := make(chan time.Time, 1)
	go func() {
		time.Sleep(delay)
		nc <- time.Now()
		for tm := range oc {
			nc <- tm
		}
	}()
	ticker.C = nc
	return ticker
}

func updateValidationCredentialsIfChanged(validationCredentialsFileName string) error {
	fileContents, err := ioutil.ReadFile(validationCredentialsFileName)
	if err != nil {
		return fmt.Errorf("could not read creds file [%v] because [%s]", envsFileName, err)
	}

	var validationCredentialsChanged bool
	var credsNewHash string
	if validationCredentialsChanged, credsNewHash, err = isFileChanged(fileContents, validationCredentialsFileName); err != nil {
		return fmt.Errorf("could not detect if creds file [%s] was changed because: [%s]", validationCredentialsFileName, err)
	}

	if !validationCredentialsChanged {
		return nil
	}

	err = updateValidationCredentials(fileContents)
	if err != nil {
		return fmt.Errorf("cannot update validation credentials because [%s]", err)
	}

	configFilesHashValues[validationCredentialsFileName] = credsNewHash
	return nil
}

func updateEnvsIfChanged(envsFileName string, envCredentialsFileName string) error {
	var envsFileChanged, envCredentialsChanged bool
	var envsNewHash, credsNewHash string

	envsfileContents, err := ioutil.ReadFile(envsFileName)
	if err != nil {
		return fmt.Errorf("could not read envs file [%s] because [%s]", envsFileName, err)
	}

	if envsFileChanged, envsNewHash, err = isFileChanged(envsfileContents, envsFileName); err != nil {
		return fmt.Errorf("could not detect if envs file [%s] was changed because [%s]", envsFileName, err)
	}

	credsFileContents, err := ioutil.ReadFile(envCredentialsFileName)
	if err != nil {
		return fmt.Errorf("could not read creds file [%s] because [%s]", envCredentialsFileName, err)
	}

	if envCredentialsChanged, credsNewHash, err = isFileChanged(credsFileContents, envCredentialsFileName); err != nil {
		return fmt.Errorf("could not detect if credentials file [%s] was changed because [%s]", envCredentialsFileName, err)
	}

	if !envsFileChanged && !envCredentialsChanged {
		return nil
	}

	err = updateEnvs(envsfileContents, credsFileContents)
	if err != nil {
		return fmt.Errorf("cannot update environments and credentials because [%s]", err)
	}
	configFilesHashValues[envsFileName] = envsNewHash
	configFilesHashValues[envCredentialsFileName] = credsNewHash
	return nil
}

func isFileChanged(contents []byte, fileName string) (bool, string, error) {
	currentHash, err := computeMD5Hash(contents)
	if err != nil {
		return false, "", fmt.Errorf("could not compute hash value for file [%s] because [%s]", fileName, err)
	}

	previousHash, found := configFilesHashValues[fileName]
	if found && previousHash == currentHash {
		return false, previousHash, nil
	}

	return true, currentHash, nil
}

func computeMD5Hash(data []byte) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("could not compute hash value because [%s]", err)
	}
	hashValue := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashValue), nil
}

func updateEnvs(envsFileData []byte, credsFileData []byte) error {
	log.Infof("Env config files changed. Updating envs")

	jsonParser := json.NewDecoder(bytes.NewReader(envsFileData))
	envsFromFile := []Environment{}
	err := jsonParser.Decode(&envsFromFile)
	if err != nil {
		return fmt.Errorf("cannot parse environmente because [%s]", err)
	}

	validEnvs := filterInvalidEnvs(envsFromFile)

	jsonParser = json.NewDecoder(bytes.NewReader(credsFileData))
	envCredentials := []Credentials{}
	err = jsonParser.Decode(&envCredentials)

	if err != nil {
		return fmt.Errorf("cannot parse credentials because [%s]", err)
	}

	environments.Lock()
	defer environments.Unlock()

	removedEnvs := parseEnvsIntoMap(validEnvs, envCredentials)
	configureFileFeeds(environments.envMap, removedEnvs)
	environments.ready = true

	return nil
}

func updateValidationCredentials(data []byte) error {
	log.Info("Updating validation credentials")

	jsonParser := json.NewDecoder(bytes.NewReader(data))
	credentials := Credentials{}
	err := jsonParser.Decode(&credentials)
	if err != nil {
		return err
	}
	validatorCredentials = credentials.Username + ":" + credentials.Password
	return nil
}

func configureFileFeeds(envMap map[string]Environment, removedEnvs []string) {
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
		for _, env := range envMap {
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
					log.Errorf("Cannot parse url [%v], error: [%v]", metric.Endpoint, err.Error())
					continue
				}

				interval := appConfig.Threshold / metric.Granularity

				if f := feeds.NewNotificationsFeed(metric.Alias, *endpointUrl, appConfig.Threshold, interval, env.Username, env.Password, metric.ApiKey); f != nil {
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
			log.Errorf("Env %v has an empty name, skipping it", env)
			continue
		}

		//envs without read-url are invalid
		if env.ReadUrl == "" {
			log.Errorf("Env with name %s does not have readUrl, skipping it", env.Name)
			continue
		}

		//envs without s3 are still valid, but still a heads up is given.
		if env.S3Url == "" {
			log.Errorf("Env with name %s does not have s3 url.", env.S3Url)
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
			log.Infof("No credentials provided for env with name %s", env.Name)
		}
	}

	//remove envs that don't exist anymore
	removedEnvs := make([]string, 0)
	for envName := range environments.envMap {
		if !isEnvInSlice(envName, envs) {
			log.Infof("removing environment from monitoring: %v", envName)
			delete(environments.envMap, envName)
			removedEnvs = append(removedEnvs, envName)
		}
	}

	//update envs
	for _, env := range envs {
		envName := env.Name
		environments.envMap[envName] = env
		log.Infof("Added environment to monitoring: %s", envName)
	}

	return removedEnvs
}

func isEnvInSlice(envName string, envs []Environment) bool {
	for _, env := range envs {
		if env.Name == envName {
			return true
		}
	}

	return false
}
