package envs

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
)

var validatorCredentials string

type Credentials struct {
	EnvName  string `json:"env-name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func WatchConfigFiles(
	wg *sync.WaitGroup,
	envsFileName, envCredentialsFileName, validationCredentialsFileName string,
	configRefreshPeriod int,
	configFilesHashValues map[string]string,
	environments *Environments,
	subscribedFeeds map[string][]feeds.Feed,
	appConfig *config.AppConfig,
	log *logger.UPPLogger,
) {
	ticker := newTicker(0, time.Minute*time.Duration(configRefreshPeriod))
	first := true
	defer func() {
		markWaitGroupDone(wg, first)
	}()

	for range ticker.C {
		err := updateEnvsIfChanged(envsFileName, envCredentialsFileName, configFilesHashValues, environments, subscribedFeeds, appConfig, log)
		if err != nil {
			log.WithError(err).Errorf("Could not update envs config")
		}

		err = updateValidationCredentialsIfChanged(validationCredentialsFileName, configFilesHashValues, log)
		if err != nil {
			log.WithError(err).Errorf("Could not update validation credentials config")
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

func updateValidationCredentialsIfChanged(validationCredentialsFileName string, configFilesHashValues map[string]string, log *logger.UPPLogger) error {
	fileContents, err := os.ReadFile(validationCredentialsFileName)
	if err != nil {
		return fmt.Errorf("could not read creds file [%v] because [%s]", validationCredentialsFileName, err)
	}

	var validationCredentialsChanged bool
	var credsNewHash string
	if validationCredentialsChanged, credsNewHash, err = isFileChanged(fileContents, validationCredentialsFileName, configFilesHashValues); err != nil {
		return fmt.Errorf("could not detect if creds file [%s] was changed because: [%s]", validationCredentialsFileName, err)
	}

	if !validationCredentialsChanged {
		return nil
	}

	err = updateValidationCredentials(fileContents, log)
	if err != nil {
		return fmt.Errorf("cannot update validation credentials because [%s]", err)
	}

	configFilesHashValues[validationCredentialsFileName] = credsNewHash
	return nil
}

func updateEnvsIfChanged(
	envsFileName, envCredentialsFileName string,
	configFilesHashValues map[string]string,
	environments *Environments,
	subscribedFeeds map[string][]feeds.Feed,
	appConfig *config.AppConfig,
	log *logger.UPPLogger,
) error {
	var envsFileChanged, envCredentialsChanged bool
	var envsNewHash, credsNewHash string

	envsfileContents, err := os.ReadFile(envsFileName)
	if err != nil {
		return fmt.Errorf("could not read envs file [%s] because [%s]", envsFileName, err)
	}

	if envsFileChanged, envsNewHash, err = isFileChanged(envsfileContents, envsFileName, configFilesHashValues); err != nil {
		return fmt.Errorf("could not detect if envs file [%s] was changed because [%s]", envsFileName, err)
	}

	credsFileContents, err := os.ReadFile(envCredentialsFileName)
	if err != nil {
		return fmt.Errorf("could not read creds file [%s] because [%s]", envCredentialsFileName, err)
	}

	if envCredentialsChanged, credsNewHash, err = isFileChanged(credsFileContents, envCredentialsFileName, configFilesHashValues); err != nil {
		return fmt.Errorf("could not detect if credentials file [%s] was changed because [%s]", envCredentialsFileName, err)
	}

	if !envsFileChanged && !envCredentialsChanged {
		return nil
	}

	err = updateEnvs(envsfileContents, credsFileContents, environments, subscribedFeeds, appConfig, log)
	if err != nil {
		return fmt.Errorf("cannot update environments and credentials because [%s]", err)
	}
	configFilesHashValues[envsFileName] = envsNewHash
	configFilesHashValues[envCredentialsFileName] = credsNewHash
	return nil
}

func isFileChanged(contents []byte, fileName string, configFilesHashValues map[string]string) (bool, string, error) {
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

func updateEnvs(envsFileData []byte, credsFileData []byte, environments *Environments, subscribedFeeds map[string][]feeds.Feed, appConfig *config.AppConfig, log *logger.UPPLogger) error {
	log.Infof("Env config files changed. Updating envs")

	jsonParser := json.NewDecoder(bytes.NewReader(envsFileData))
	envsFromFile := []Environment{}
	err := jsonParser.Decode(&envsFromFile)
	if err != nil {
		return fmt.Errorf("cannot parse environmente because [%s]", err)
	}

	validEnvs := filterInvalidEnvs(envsFromFile, log)

	jsonParser = json.NewDecoder(bytes.NewReader(credsFileData))
	envCredentials := []Credentials{}
	err = jsonParser.Decode(&envCredentials)

	if err != nil {
		return fmt.Errorf("cannot parse credentials because [%s]", err)
	}

	removedEnvs := parseEnvsIntoMap(validEnvs, envCredentials, environments, log)
	configureFileFeeds(environments.Values(), removedEnvs, subscribedFeeds, appConfig, log)
	environments.SetReady(true)

	return nil
}

func updateValidationCredentials(data []byte, log *logger.UPPLogger) error {
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

func configureFileFeeds(envs []Environment, removedEnvs []string, subscribedFeeds map[string][]feeds.Feed, appConfig *config.AppConfig, log *logger.UPPLogger) {
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
		for _, env := range envs {
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
				endpointURL, err := url.Parse(env.ReadURL + metric.Endpoint)
				if err != nil {
					log.WithError(err).Errorf("Cannot parse url [%v]", metric.Endpoint)
					continue
				}

				interval := appConfig.Threshold / metric.Granularity

				if f := feeds.NewNotificationsFeed(metric.Alias, *endpointURL, appConfig.Threshold, interval, env.Username, env.Password, metric.APIKey, log); f != nil {
					subscribedFeeds[env.Name] = append(envFeeds, f)
					f.Start()
				}
			}
		}
	}
}

func filterInvalidEnvs(envsFromFile []Environment, log *logger.UPPLogger) []Environment {
	var validEnvs []Environment
	for _, env := range envsFromFile {
		//envs without name are invalid
		if env.Name == "" {
			log.Errorf("Env %v has an empty name, skipping it", env)
			continue
		}

		//envs without read-url are invalid
		if env.ReadURL == "" {
			log.Errorf("Env with name %s does not have readUrl, skipping it", env.Name)
			continue
		}

		validEnvs = append(validEnvs, env)
	}

	return validEnvs
}

func parseEnvsIntoMap(envs []Environment, envCredentials []Credentials, environments *Environments, log *logger.UPPLogger) []string {
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
	envNames := environments.Names()
	for _, envName := range envNames {
		if !isEnvInSlice(envName, envs) {
			log.Infof("removing environment from monitoring: %v", envName)
			environments.RemoveEnvironment(envName)
			removedEnvs = append(removedEnvs, envName)
		}
	}

	//update envs
	for _, env := range envs {
		envName := env.Name
		environments.SetEnvironment(envName, env)
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

func GetValidationCredentials() (string, string) {
	if strings.Contains(validatorCredentials, ":") {
		unpw := strings.SplitN(validatorCredentials, ":", 2)
		return unpw[0], unpw[1]
	}

	return "", ""
}
