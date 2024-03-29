package envs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/stretchr/testify/assert"
)

const (
	validEnvConfig = `
		[
			{
				"name":"test-env",
				"read-url": "https://test-env.ft.com"
			}
		]`
	//nolint:gosec
	validEnvCredentialsConfig = `
		[
			{
				"env-name": "test-env",
				"username": "test-user",
				"password": "test-pwd"
			}
		]`
	//nolint:gosec
	validValidationCredentialsConfig = `
		{
			"username": "test-user",
			"password": "test-pwd"
		}`
	invalidJSONConfig = `invalid-config`
)

func TestParseEnvsIntoMap(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := getValidCredentials()
	environments := NewEnvironments()
	log := logger.NewUPPLogger("test", "PANIC")

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials, environments, log)

	assert.Equal(t, 0, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.Len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.Environment(envName).Name)
	assert.Equal(t, credentials[1].Username, environments.Environment(envName).Username)
}

func TestParseEnvsIntoMapWithRemovedEnv(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := getValidCredentials()
	environments := NewEnvironments()
	environments.SetEnvironment("removed-env", Environment{})
	log := logger.NewUPPLogger("test", "PANIC")

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials, environments, log)

	assert.Equal(t, 1, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.Len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.Environment(envName).Name)
	assert.Equal(t, credentials[1].Username, environments.Environment(envName).Username)
}

func TestParseEnvsIntoMapWithExistingEnv(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := getValidCredentials()
	environments := NewEnvironments()
	existingEnv := envsToBeParsed[0]
	environments.SetEnvironment(existingEnv.Name, existingEnv)
	log := logger.NewUPPLogger("test", "PANIC")

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials, environments, log)

	assert.Equal(t, 0, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.Len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.Environment(envName).Name)
	assert.Equal(t, credentials[1].Username, environments.Environment(envName).Username)
}

func TestParseEnvsIntoMapWithNoCredentials(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := []Credentials{}
	environments := NewEnvironments()
	log := logger.NewUPPLogger("test", "PANIC")

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials, environments, log)

	assert.Equal(t, 0, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.Len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.Environment(envName).Name)
}

func TestFilterInvalidEnvs(t *testing.T) {
	envsToBeFiltered := getValidEnvs()
	log := logger.NewUPPLogger("test", "PANIC")

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered, log)

	assert.Equal(t, len(envsToBeFiltered), len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyName(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "",
			ReadURL:  "test",
			Username: "dummy",
			Password: "dummy",
		},
	}
	log := logger.NewUPPLogger("test", "PANIC")

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered, log)

	assert.Equal(t, 0, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyReadUrl(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadURL:  "",
			Username: "dummy",
			Password: "dummy",
		},
	}
	log := logger.NewUPPLogger("test", "PANIC")

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered, log)

	assert.Equal(t, 0, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyUsernameUrl(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadURL:  "test",
			Username: "",
			Password: "dummy",
		},
	}
	log := logger.NewUPPLogger("test", "PANIC")

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered, log)

	assert.Equal(t, 1, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyPwd(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadURL:  "test",
			Username: "test",
			Password: "",
		},
	}
	log := logger.NewUPPLogger("test", "PANIC")

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered, log)

	assert.Equal(t, 1, len(filteredEnvs))
}

func TestUpdateValidationCredentialsHappyFlow(t *testing.T) {
	log := logger.NewUPPLogger("test", "PANIC")

	fileName := prepareFile(validValidationCredentialsConfig)
	fileContents, _ := os.ReadFile(fileName)
	err := updateValidationCredentials(fileContents, log)

	assert.Nil(t, err)
	assert.Equal(t, "test-user:test-pwd", validatorCredentials)
	os.Remove(fileName)
}

func TestUpdateValidationCredentialNilFile(t *testing.T) {
	validatorCredentials := Credentials{
		Username: "test-username",
		Password: "test-password",
	}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateValidationCredentials(nil, log)

	assert.NotNil(t, err)
	//make sure validationCredentials didn't change after failing call to updateValidationCredentials().
	assert.Equal(t, "test-username", validatorCredentials.Username)
	assert.Equal(t, "test-password", validatorCredentials.Password)
}

func TestUpdateValidationCredentialsInvalidConfig(t *testing.T) {
	fileName := prepareFile(invalidJSONConfig)
	validatorCredentials := Credentials{
		Username: "test-username",
		Password: "test-password",
	}
	fileContents, _ := os.ReadFile(fileName)
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateValidationCredentials(fileContents, log)
	assert.NotNil(t, err)
	//make sure validationCredentials didn't change after failing call to updateValidationCredentials().
	assert.Equal(t, "test-username", validatorCredentials.Username)
	assert.Equal(t, "test-password", validatorCredentials.Password)
	os.Remove(fileName)
}

func TestConfigureFeedsWithEmptyListOfMetrics(t *testing.T) {
	subscribedFeeds := map[string][]feeds.Feed{}
	subscribedFeeds["test-feed"] = []feeds.Feed{
		MockFeed{},
	}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	configureFileFeeds(make([]Environment, 0), []string{"test-feed"}, subscribedFeeds, appConfig, log)

	assert.Equal(t, 0, len(subscribedFeeds))
}

func TestUpdateEnvsHappyFlow(t *testing.T) {
	subscribedFeeds := map[string][]feeds.Feed{}
	subscribedFeeds["test-feed"] = []feeds.Feed{
		MockFeed{},
	}
	appConfig := &config.AppConfig{}
	envsFileName := prepareFile(validEnvConfig)
	envsFileContents, _ := os.ReadFile(envsFileName)

	envCredsFileName := prepareFile(validEnvCredentialsConfig)
	credsFileContents, _ := os.ReadFile(envCredsFileName)
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvs(envsFileContents, credsFileContents, NewEnvironments(), subscribedFeeds, appConfig, log)

	assert.Nil(t, err)
	os.Remove(envsFileName)
	os.Remove(envCredsFileName)
}

func TestUpdateEnvsHappyNilEnvsFile(t *testing.T) {
	envCredsFileName := prepareFile(validEnvCredentialsConfig)
	credsFileContents, _ := os.ReadFile(envCredsFileName)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvs(nil, credsFileContents, NewEnvironments(), subscribedFeeds, appConfig, log)

	assert.NotNil(t, err)
	os.Remove(envCredsFileName)
}

func TestUpdateEnvsNilEnvCredentialsFile(t *testing.T) {
	envsFileName := prepareFile(validEnvConfig)
	envsFileContents, _ := os.ReadFile(envsFileName)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvs(envsFileContents, nil, NewEnvironments(), subscribedFeeds, appConfig, log)

	assert.NotNil(t, err)
	os.Remove(envsFileName)
}

func TestComputeMD5Hash(t *testing.T) {
	var testCases = []struct {
		caseDescription string
		toHash          []byte
		expectedHash    string
	}{
		{
			caseDescription: "one-line valid input",
			toHash:          []byte("foobar"),
			expectedHash:    "3858f62230ac3c915f300c664312c63f",
		},
		{
			caseDescription: "multi-line valid input",
			toHash: []byte(`foo
					      bar`),
			expectedHash: "1be7783a9859a16a010d466d39342543",
		},
		{
			caseDescription: "empty input",
			toHash:          []byte(""),
			expectedHash:    "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			caseDescription: "nil input",
			toHash:          nil,
			expectedHash:    "d41d8cd98f00b204e9800998ecf8427e",
		},
	}

	for _, tc := range testCases {
		actualHash, err := computeMD5Hash(tc.toHash)
		assert.Nil(t, err)
		assert.Equal(t, tc.expectedHash, actualHash,
			fmt.Sprintf("%s: Computed has doesn't match expected hash", tc.caseDescription))
	}
}

func TestIsFileChanged(t *testing.T) {
	var testCases = []struct {
		caseDescription       string
		fileContents          []byte
		fileName              string
		configFilesHashValues map[string]string
		expectedResult        bool
		expectedHash          string
	}{
		{
			caseDescription: "file not changed",
			fileContents:    []byte("foobar"),
			fileName:        "file1",
			configFilesHashValues: map[string]string{
				"file1": "3858f62230ac3c915f300c664312c63f",
				"file2": "1be7783a9859a16a010d466d39342543",
			},
			expectedResult: false,
			expectedHash:   "3858f62230ac3c915f300c664312c63f",
		},
		{
			caseDescription: "new file",
			fileContents:    []byte("foobar"),
			fileName:        "file1",
			configFilesHashValues: map[string]string{
				"file2": "1be7783a9859a16a010d466d39342543",
			},
			expectedResult: true,
			expectedHash:   "3858f62230ac3c915f300c664312c63f",
		},
		{
			caseDescription: "file contents changed",
			fileContents:    []byte("foobarNew"),
			fileName:        "file1",
			configFilesHashValues: map[string]string{
				"file1": "3858f62230ac3c915f300c664312c63f",
				"file2": "1be7783a9859a16a010d466d39342543",
			},
			expectedResult: true,
			expectedHash:   "bdcf75c01270b40ebb33c1d24457ed81",
		},
	}

	for _, tc := range testCases {
		configFilesHashValues := tc.configFilesHashValues
		actualResult, actualHash, _ := isFileChanged(tc.fileContents, tc.fileName, configFilesHashValues)
		assert.Equal(t, tc.expectedResult, actualResult,
			fmt.Sprintf("%s: File change was not detected correctly.", tc.caseDescription))
		assert.Equal(t, tc.expectedHash, actualHash,
			fmt.Sprintf("%s: The expected file hash was not returned.", tc.caseDescription))
	}
}

func TestUpdateEnvsIfChangedEnvFileDoesntExist(t *testing.T) {
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged("thisFileDoesntexist", credsFile, configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, environments.Len(), "No new environments should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateEnvsIfChangedCredsFileDoesntExist(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)

	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged(envsFile, "thisFileDoesntexist", configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, environments.Len(), "No new environments should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateEnvsIfChangedFilesDontExist(t *testing.T) {
	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged("thisFileDoesntExist", "thisDoesntExistEither", configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.NotNil(t, err, "Didn't get an error after supplying files which don't exist")
	assert.Equal(t, 0, environments.Len(), "No new environments should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateEnvsIfChangedValidFiles(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}

	//appConfig has to be non-nil for the actual update to work
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged(envsFile, credsFile, configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.Nil(t, err, "Got an error after supplying valid files")
	assert.Equal(t, 1, environments.Len(), "New environment should've been added")
	assert.Equal(t, 2, len(configFilesHashValues), "New hashes should've been added")
}

func TestUpdateEnvsIfChangedNoChanges(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	subscribedFeeds := map[string][]feeds.Feed{}
	environments := NewEnvironments()
	environments.SetEnvironment("test-env", Environment{
		Name:     "test-env",
		Password: "test-pwd",
		ReadURL:  "https://test-env.ft.com",
		Username: "test-user",
	})

	configFilesHashValues := map[string]string{
		envsFile:  "aeb7d7ba7e2169de3552165c4c2d5571",
		credsFile: "dfd8aecc21b7017c5e4f171e3279fc68",
	}

	//if the update works (which it shouldn't) we will have a failure
	var appConfig *config.AppConfig
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged(envsFile, credsFile, configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.Nil(t, err, "Got an error after supplying valid files")
	assert.Equal(t, 1, environments.Len(), "Environments shouldn't have changed")
	assert.Equal(t, 2, len(configFilesHashValues), "Hashes shouldn't have changed")
}

func TestUpdateEnvsIfChangedInvalidEnvsFile(t *testing.T) {
	envsFile := prepareFile(invalidJSONConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged(envsFile, credsFile, configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.NotNil(t, err, "Didn't get an error after supplying invalid file")
	assert.Equal(t, 0, environments.Len(), "No new environment should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No new hashes should've been added")
}

func TestUpdateEnvsIfChangedInvalidCredsFile(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(invalidJSONConfig)
	defer os.Remove(credsFile)

	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged(envsFile, credsFile, configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.NotNil(t, err, "Didn't get an error after supplying invalid file")
	assert.Equal(t, 0, environments.Len(), "No new environment should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No new hashes should've been added")
}

func TestUpdateEnvsIfChangedInvalidFiles(t *testing.T) {
	envsFile := prepareFile(invalidJSONConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(invalidJSONConfig)
	defer os.Remove(credsFile)

	environments := NewEnvironments()
	configFilesHashValues := make(map[string]string)
	subscribedFeeds := map[string][]feeds.Feed{}
	appConfig := &config.AppConfig{}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateEnvsIfChanged(envsFile, credsFile, configFilesHashValues, environments, subscribedFeeds, appConfig, log)

	assert.NotNil(t, err, "Didn't get an error after supplying invalid file")
	assert.Equal(t, 0, environments.Len(), "No new environment should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No new hashes should've been added")
}

func TestUpdateValidationCredentialsIfChangedFileDoesntExist(t *testing.T) {
	validatorCredentials = ""
	configFilesHashValues := make(map[string]string)
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateValidationCredentialsIfChanged("thisFileDoesntExist", configFilesHashValues, log)

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, len(validatorCredentials), "No validator credentials should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateValidationCredentialsIfChangedInvalidFile(t *testing.T) {
	validationCredsFile := prepareFile(invalidJSONConfig)
	defer os.Remove(validationCredsFile)

	validatorCredentials = ""
	configFilesHashValues := make(map[string]string)
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateValidationCredentialsIfChanged(validationCredsFile, configFilesHashValues, log)

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, len(validatorCredentials), "No validator credentials should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateValidationCredentialsIfChangedNewFile(t *testing.T) {
	validationCredsFile := prepareFile(validValidationCredentialsConfig)
	defer os.Remove(validationCredsFile)

	validatorCredentials = ""
	configFilesHashValues := make(map[string]string)
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateValidationCredentialsIfChanged(validationCredsFile, configFilesHashValues, log)

	assert.Nil(t, err, "Shouldn't get an error for valid file")
	assert.Equal(t, "test-user:test-pwd", validatorCredentials, "New validator credentials should've been added")
	assert.Equal(t, 1, len(configFilesHashValues), "New hashes should've been added")
}

func TestUpdateValidationCredentialsIfChangedFileUnchanged(t *testing.T) {
	validationCredsFile := prepareFile(validValidationCredentialsConfig)
	defer os.Remove(validationCredsFile)

	validatorCredentials = "test-user:test-pwd"
	configFilesHashValues := map[string]string{
		validationCredsFile: "cc4d51dfe137ec8cbba8fd3ff24474be",
	}
	log := logger.NewUPPLogger("test", "PANIC")

	err := updateValidationCredentialsIfChanged(validationCredsFile, configFilesHashValues, log)
	assert.Nil(t, err, "Shouldn't get an error for valid file")
	assert.Equal(t, "test-user:test-pwd", validatorCredentials, "Validator credentials shouldn't have changed")
	assert.Equal(t, "cc4d51dfe137ec8cbba8fd3ff24474be", configFilesHashValues[validationCredsFile], "Hashes shouldn't have changed")
}

func TestTickerWithInitialDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	delay := 2
	ticker := newTicker(time.Duration(delay)*time.Second, time.Minute)
	defer ticker.Stop()

	before := time.Now()
	go func() {
		<-ticker.C
		cancel()
	}()

	select {
	case <-ctx.Done():
		assert.WithinDuration(t, before.Add(time.Duration(delay)*time.Second), time.Now(), time.Second, "initial tick")
	case <-time.After(time.Duration(delay+1) * time.Second):
		assert.Fail(t, "timed out waiting for initial tick")
	}
}

func prepareFile(fileContent string) string {
	file, err := os.CreateTemp(os.TempDir(), "")
	if err != nil {
		panic("Cannot create temp file.")
	}

	writer := bufio.NewWriter(file)
	defer file.Close()
	fmt.Fprintln(writer, fileContent)
	writer.Flush()
	return file.Name()
}

func getValidEnvs() []Environment {
	return []Environment{
		{
			Name:    "test",
			ReadURL: "test-url",
		},
		{
			Name:    "test2",
			ReadURL: "test-url2",
		},
	}
}

func getValidCredentials() []Credentials {
	return []Credentials{
		{
			EnvName:  "test",
			Username: "dummy-user",
			Password: "dummy-pwd",
		},
		{
			EnvName:  "test2",
			Username: "dummy-user2",
			Password: "dummy-pwd2",
		},
	}
}

type MockFeed struct{}

func (f MockFeed) Start() {}
func (f MockFeed) Stop()  {}
func (f MockFeed) FeedName() string {
	return ""
}
func (f MockFeed) FeedURL() string {
	return ""
}
func (f MockFeed) FeedType() string {
	return ""
}
func (f MockFeed) SetCredentials(username string, password string) {}
func (f MockFeed) NotificationsFor(uuid string) []*feeds.Notification {
	return nil
}
